import 'whatwg-fetch';

export const ApiHelpers = pathPrefix => {
  let metricsWindow = '1m';
  const podsPath = `${pathPrefix}/api/pods`;
  const metricsPath = () => `${pathPrefix}/api/metrics?window=${metricsWindow}`;

  const validMetricsWindows = {
    "10s": true,
    "1m": true,
    "10m": true
  };

  const apiFetch = path => {
    return fetch(path).then(handleFetchErr).then(r => r.json());
  };

  const fetchMetrics = (path = metricsPath()) => {
    if (path.indexOf("window") === -1) {
      path = `${path}&window=${getMetricsWindow()}`;
    }
    return apiFetch(path);
  };

  const fetchPods = () => {
    return apiFetch(podsPath);
  };

  const handleFetchErr = resp => {
    if (!resp.ok) {
      throw Error(resp.statusText);
    }
    return resp;
  };

  const getMetricsWindow = () => metricsWindow;

  const setMetricsWindow = window => {
    if (!validMetricsWindows[window]) return;
    metricsWindow = window;
  };

  const metricsUrl = `${pathPrefix}/api/metrics?`;
  const urlsForResource = {
    // all deploys (default), or a given deploy if specified
    "deployment": {
      groupBy: "targetDeploy",
      url: (deploy = null) => {
        let rollupUrl = !deploy ? metricsUrl : `${metricsUrl}&target_deploy=${deploy}`;
        let timeseriesUrl = !deploy ? `${metricsUrl}&timeseries=true` :
          `${metricsUrl}&timeseries=true&target_deploy=${deploy}`;
        return {
          ts: timeseriesUrl,
          rollup: rollupUrl
        };
      }
    },
    "pod": {
      // all pods of a given deploy
      groupBy: "targetPod",
      url: deploy => {
        let podRollupUrl = `${metricsUrl}&aggregation=target_pod&target_deploy=${deploy}`;
        let podTimeseriesUrl = `${podRollupUrl}&timeseries=true`;
        return {
          ts: podTimeseriesUrl,
          rollup: podRollupUrl
        };
      }
    },
    "upstream_deployment": {
      // all upstreams of a given deploy
      groupBy: "sourceDeploy",
      url: deploy => {
        let upstreamRollupUrl = `${metricsUrl}&aggregation=source_deploy&target_deploy=${deploy}`;
        let upstreamTimeseriesUrl = `${upstreamRollupUrl}&timeseries=true`;
        return {
          ts: upstreamTimeseriesUrl,
          rollup: upstreamRollupUrl
        };
      }
    },
    "downstream_deployment": {
      // all downstreams of a given deploy
      groupBy: "targetDeploy",
      url: deploy => {
        let downstreamRollupUrl = `${metricsUrl}&aggregation=target_deploy&source_deploy=${deploy}`;
        let downstreamTimeseriesUrl = `${downstreamRollupUrl}&timeseries=true`;
        return {
          ts: downstreamTimeseriesUrl,
          rollup: downstreamRollupUrl
        };
      }
    },
    "upstream_pod": {
      groupBy: "sourcePod",
      url: pod => {
        let upstreamRollupUrl = `${metricsUrl}&aggregation=source_pod&target_pod=${pod}`;
        let upstreamTimeseriesUrl = `${upstreamRollupUrl}&timeseries=true`;

        return {
          ts: upstreamTimeseriesUrl,
          rollup: upstreamRollupUrl
        };
      }
    },
    "downstream_pod": {
      groupBy: "targetPod",
      url: pod => {
        let downstreamRollupUrl = `${metricsUrl}&aggregation=target_pod&source_pod=${pod}`;
        let downstreamTimeseriesUrl = `${downstreamRollupUrl}&timeseries=true`;

        return {
          ts: downstreamTimeseriesUrl,
          rollup: downstreamRollupUrl
        };
      }
    },
    "path": {
      // all paths (default), or all paths of a given deploy if specified
      groupBy: "path",
      url: (deploy = null) => {
        let pathRollupUrl = `${metricsUrl}&aggregation=path${ !deploy ? "" : `&target_deploy=${deploy}`}`;
        let pathTsUrl = `${pathRollupUrl}&timeseries=true`;

        return {
          ts: pathTsUrl,
          rollup: pathRollupUrl
        };
      }
    }
  };

  return {
    fetch: apiFetch,
    fetchMetrics,
    fetchPods,
    getMetricsWindow,
    setMetricsWindow,
    urlsForResource: urlsForResource
  };
};
