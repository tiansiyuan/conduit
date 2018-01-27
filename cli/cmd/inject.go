package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/runconduit/conduit/pkg/k8s"
	"github.com/runconduit/conduit/pkg/version"
	"github.com/spf13/cobra"
	batchV1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yamlDecoder "k8s.io/apimachinery/pkg/util/yaml"
)

var (
	initImage           string
	proxyImage          string
	proxyUID            int64
	inboundPort         uint
	outboundPort        uint
	ignoreInboundPorts  []uint
	ignoreOutboundPorts []uint
	proxyControlPort    uint
	proxyAPIPort        uint
)

var injectCmd = &cobra.Command{
	Use:   "inject [flags] CONFIG-FILE",
	Short: "Add the Conduit proxy to a Kubernetes config",
	Long: `Add the Conduit proxy to a Kubernetes config.

You can use a config file from stdin by using the '-' argument
with 'conduit inject'. e.g. curl http://url.to/yml | conduit inject -
	`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if len(args) < 1 {
			return fmt.Errorf("please specify a deployment file")
		}

		var in io.Reader
		var err error

		if args[0] == "-" {
			in = os.Stdin
		} else {
			if in, err = os.Open(args[0]); err != nil {
				return err
			}
		}

		reader := yamlDecoder.NewYAMLReader(bufio.NewReaderSize(in, 4096))
		// Iterate over all YAML objects in the input
		for {
			// Read a single YAML object
			bytes, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			// Unmarshal the object enough to read the Kind field
			var meta metaV1.TypeMeta
			if err := yaml.Unmarshal(bytes, &meta); err != nil {
				return err
			}

			var injected interface{} = nil
			switch meta.Kind {
			case "Deployment":
				injected, err = injectDeployment(bytes)
			case "ReplicationController":
				injected, err = injectReplicationController(bytes)
			case "ReplicaSet":
				injected, err = injectReplicaSet(bytes)
			case "Job":
				injected, err = injectJob(bytes)
			case "DaemonSet":
				injected, err = injectDaemonSet(bytes)
			}
			output := bytes
			if injected != nil {
				output, err = yaml.Marshal(injected)
				if err != nil {
					return err
				}
			}
			os.Stdout.Write(output)
			fmt.Println("---")
		}
		return nil
	},
}

/* Given a byte slice representing a deployment, unmarshal the deployment and
 * return a new deployment with the sidecar and init-container injected.
 */
func injectDeployment(bytes []byte) (interface{}, error) {
	var deployment v1beta1.Deployment
	err := yaml.Unmarshal(bytes, &deployment)
	if err != nil {
		return nil, err
	}
	podTemplateSpec := injectPodTemplateSpec(&deployment.Spec.Template)
	return enhancedDeployment{
		&deployment,
		enhancedDeploymentSpec{
			&deployment.Spec,
			podTemplateSpec,
		},
	}, nil
}

/* Given a byte slice representing a replication controller, unmarshal the
 * replication controller and return a new replication controller with the
 * sidecar and init-container injected.
 */
func injectReplicationController(bytes []byte) (interface{}, error) {
	var rc v1.ReplicationController
	err := yaml.Unmarshal(bytes, &rc)
	if err != nil {
		return nil, err
	}
	podTemplateSpec := injectPodTemplateSpec(rc.Spec.Template)
	return enhancedReplicationController{
		&rc,
		enhancedReplicationControllerSpec{
			&rc.Spec,
			podTemplateSpec,
		},
	}, nil
}

/* Given a byte slice representing a replica set, unmarshal the replica set and
 * return a new replica set with the sidecar and init-container injected.
 */
func injectReplicaSet(bytes []byte) (interface{}, error) {
	var rs v1beta1.ReplicaSet
	err := yaml.Unmarshal(bytes, &rs)
	if err != nil {
		return nil, err
	}
	podTemplateSpec := injectPodTemplateSpec(&rs.Spec.Template)
	return enhancedReplicaSet{
		&rs,
		enhancedReplicaSetSpec{
			&rs.Spec,
			podTemplateSpec,
		},
	}, nil
}

/* Given a byte slice representing a job, unmarshal the job and return a new job
 * with the sidecar and init-container injected.
 */
func injectJob(bytes []byte) (interface{}, error) {
	var job batchV1.Job
	err := yaml.Unmarshal(bytes, &job)
	if err != nil {
		return nil, err
	}
	podTemplateSpec := injectPodTemplateSpec(&job.Spec.Template)
	return enhancedJob{
		&job,
		enhancedJobSpec{
			&job.Spec,
			podTemplateSpec,
		},
	}, nil
}

/* Given a byte slice representing a daemonset, unmarshal the daemonset and
 * return a new daemonset with the sidecar and init-container injected.
 */
func injectDaemonSet(bytes []byte) (interface{}, error) {
	var ds v1beta1.DaemonSet
	err := yaml.Unmarshal(bytes, &ds)
	if err != nil {
		return nil, err
	}
	podTemplateSpec := injectPodTemplateSpec(&ds.Spec.Template)
	return enhancedDaemonSet{
		&ds,
		enhancedDaemonSetSpec{
			&ds.Spec,
			podTemplateSpec,
		},
	}, nil
}

/* Given a PodTemplateSpec, return a new PodTemplateSpec with the sidecar
 * and init-container injected.
 */
func injectPodTemplateSpec(t *v1.PodTemplateSpec) enhancedPodTemplateSpec {
	f := false
	inboundSkipPorts := append(ignoreInboundPorts, proxyControlPort)
	inboundSkipPortsStr := make([]string, len(inboundSkipPorts))
	for i, p := range inboundSkipPorts {
		inboundSkipPortsStr[i] = strconv.Itoa(int(p))
	}

	outboundSkipPortsStr := make([]string, len(ignoreOutboundPorts))
	for i, p := range ignoreOutboundPorts {
		outboundSkipPortsStr[i] = strconv.Itoa(int(p))
	}

	initArgs := []string{
		"--incoming-proxy-port", fmt.Sprintf("%d", inboundPort),
		"--outgoing-proxy-port", fmt.Sprintf("%d", outboundPort),
		"--proxy-uid", fmt.Sprintf("%d", proxyUID),
	}

	if len(inboundSkipPortsStr) > 0 {
		initArgs = append(initArgs, "--inbound-ports-to-ignore")
		initArgs = append(initArgs, strings.Join(inboundSkipPortsStr, ","))
	}

	if len(outboundSkipPortsStr) > 0 {
		initArgs = append(initArgs, "--outbound-ports-to-ignore")
		initArgs = append(initArgs, strings.Join(outboundSkipPortsStr, ","))
	}

	initContainer := v1.Container{
		Name:            "conduit-init",
		Image:           fmt.Sprintf("%s:%s", initImage, conduitVersion),
		ImagePullPolicy: v1.PullPolicy(imagePullPolicy),
		Args:            initArgs,
		SecurityContext: &v1.SecurityContext{
			Capabilities: &v1.Capabilities{
				Add: []v1.Capability{v1.Capability("NET_ADMIN")},
			},
			Privileged: &f,
		},
	}

	sidecar := v1.Container{
		Name:            "conduit-proxy",
		Image:           fmt.Sprintf("%s:%s", proxyImage, conduitVersion),
		ImagePullPolicy: v1.PullPolicy(imagePullPolicy),
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &proxyUID,
		},
		Ports: []v1.ContainerPort{
			v1.ContainerPort{
				Name:          "conduit-proxy",
				ContainerPort: int32(inboundPort),
			},
		},
		Env: []v1.EnvVar{
			v1.EnvVar{Name: "CONDUIT_PROXY_LOG", Value: "warn,conduit_proxy=info"},
			v1.EnvVar{
				Name:  "CONDUIT_PROXY_CONTROL_URL",
				Value: fmt.Sprintf("tcp://proxy-api.%s.svc.cluster.local:%d", controlPlaneNamespace, proxyAPIPort),
			},
			v1.EnvVar{Name: "CONDUIT_PROXY_CONTROL_LISTENER", Value: fmt.Sprintf("tcp://0.0.0.0:%d", proxyControlPort)},
			v1.EnvVar{Name: "CONDUIT_PROXY_PRIVATE_LISTENER", Value: fmt.Sprintf("tcp://127.0.0.1:%d", outboundPort)},
			v1.EnvVar{Name: "CONDUIT_PROXY_PUBLIC_LISTENER", Value: fmt.Sprintf("tcp://0.0.0.0:%d", inboundPort)},
			v1.EnvVar{
				Name:      "CONDUIT_PROXY_NODE_NAME",
				ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "spec.nodeName"}},
			},
			v1.EnvVar{
				Name:      "CONDUIT_PROXY_POD_NAME",
				ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.name"}},
			},
			v1.EnvVar{
				Name:      "CONDUIT_PROXY_POD_NAMESPACE",
				ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
			},
			v1.EnvVar{
				Name:  "CONDUIT_PROXY_DESTINATIONS_AUTOCOMPLETE_FQDN",
				Value: "Kubernetes",
			},
		},
	}

	if t.Annotations == nil {
		t.Annotations = make(map[string]string)
	}
	t.Annotations[k8s.CreatedByAnnotation] = fmt.Sprintf("conduit/cli %s", version.Version)
	t.Annotations[k8s.ProxyVersionAnnotation] = conduitVersion

	if t.Labels == nil {
		t.Labels = make(map[string]string)
	}
	t.Labels[k8s.ControllerNSLabel] = controlPlaneNamespace
	t.Spec.Containers = append(t.Spec.Containers, sidecar)
	return enhancedPodTemplateSpec{
		t,
		enhancedPodSpec{
			&t.Spec,
			append(t.Spec.InitContainers, initContainer),
		},
	}
}

/* The v1.PodSpec struct contains a field annotation that causes the
 * InitContainers field to be omitted when serializing the struct as json.
 * Since we wish for this field to be included, we have to define our own
 * enhancedPodSpec struct with a different annotation on this field.  We then
 * must define our own structs to use this struct, and so on.
 */
type enhancedPodSpec struct {
	*v1.PodSpec
	InitContainers []v1.Container `json:"initContainers"`
}

type enhancedPodTemplateSpec struct {
	*v1.PodTemplateSpec
	Spec enhancedPodSpec `json:"spec,omitempty"`
}

type enhancedDeploymentSpec struct {
	*v1beta1.DeploymentSpec
	Template enhancedPodTemplateSpec `json:"template,omitempty"`
}

type enhancedDeployment struct {
	*v1beta1.Deployment
	Spec enhancedDeploymentSpec `json:"spec,omitempty"`
}

type enhancedReplicationControllerSpec struct {
	*v1.ReplicationControllerSpec
	Template enhancedPodTemplateSpec `json:"template,omitempty"`
}

type enhancedReplicationController struct {
	*v1.ReplicationController
	Spec enhancedReplicationControllerSpec `json:"spec,omitempty"`
}

type enhancedReplicaSetSpec struct {
	*v1beta1.ReplicaSetSpec
	Template enhancedPodTemplateSpec `json:"template,omitempty"`
}

type enhancedReplicaSet struct {
	*v1beta1.ReplicaSet
	Spec enhancedReplicaSetSpec `json:"spec,omitempty"`
}

type enhancedJobSpec struct {
	*batchV1.JobSpec
	Template enhancedPodTemplateSpec `json:"template,omitempty"`
}

type enhancedJob struct {
	*batchV1.Job
	Spec enhancedJobSpec `json:"spec,omitempty"`
}

type enhancedDaemonSetSpec struct {
	*v1beta1.DaemonSetSpec
	Template enhancedPodTemplateSpec `json:"template,omitempty"`
}

type enhancedDaemonSet struct {
	*v1beta1.DaemonSet
	Spec enhancedDaemonSetSpec `json:"spec,omitempty"`
}

func init() {
	RootCmd.AddCommand(injectCmd)
	injectCmd.PersistentFlags().StringVarP(&conduitVersion, "conduit-version", "v", version.Version, "tag to be used for conduit images")
	injectCmd.PersistentFlags().StringVar(&initImage, "init-image", "gcr.io/runconduit/proxy-init", "Conduit init container image name")
	injectCmd.PersistentFlags().StringVar(&proxyImage, "proxy-image", "gcr.io/runconduit/proxy", "Conduit proxy container image name")
	injectCmd.PersistentFlags().StringVar(&imagePullPolicy, "image-pull-policy", "IfNotPresent", "Docker image pull policy")
	injectCmd.PersistentFlags().Int64Var(&proxyUID, "proxy-uid", 2102, "Run the proxy under this user ID")
	injectCmd.PersistentFlags().UintVar(&inboundPort, "inbound-port", 4143, "proxy port to use for inbound traffic")
	injectCmd.PersistentFlags().UintVar(&outboundPort, "outbound-port", 4140, "proxy port to use for outbound traffic")
	injectCmd.PersistentFlags().UintSliceVar(&ignoreInboundPorts, "skip-inbound-ports", nil, "ports that should skip the proxy and send directly to the applicaiton")
	injectCmd.PersistentFlags().UintSliceVar(&ignoreOutboundPorts, "skip-outbound-ports", nil, "outbound ports that should skip the proxy")
	injectCmd.PersistentFlags().UintVar(&proxyControlPort, "control-port", 4190, "proxy port to use for control")
	injectCmd.PersistentFlags().UintVar(&proxyAPIPort, "api-port", 8086, "port where the Conduit controller is running")
}
