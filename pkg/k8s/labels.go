/*
Kubernetes labels and annotations used in Conduit's control plane and data plane
Kubernetes configs.
*/

package k8s

const (
	/*
	 * Labels
	 */

	// ControllerComponentLabel identifies this object as a component of Conduit's
	// control plane (e.g. web, controller).
	ControllerComponentLabel = "conduit.io/controller-component"

	// ControllerNSLabel is injected into mesh-enabled apps, identifying the
	// namespace of the Conduit control plane.
	ControllerNSLabel = "conduit.io/controller-ns"

	/*
	 * Annotations
	 */

	// CreatedByAnnotation indicates the source of the injected data plane
	// (e.g. conduit/cli v0.1.3).
	CreatedByAnnotation = "conduit.io/created-by"

	// ProxyVersionAnnotation indicates the version of the injected data plane
	// (e.g. v0.1.3)
	ProxyVersionAnnotation = "conduit.io/proxy-version"
)
