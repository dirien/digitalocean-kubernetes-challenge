package charts

import (
	"fmt"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewArgoCD(ctx *pulumi.Context, args CreateChartArgs, parent *helm.Release) (*helm.Release, error) {

	oidcString := pulumi.All(args.Auth.GetStringOutput(pulumi.String("argo.clientId")),
		args.Auth.GetStringOutput(pulumi.String("argo.clientSecret"))).ApplyT(
		func(args []interface{}) (string, error) {
			clientId := args[0].(string)
			clientSecret := args[1].(string)
			return fmt.Sprintf(`name: Auth0
issuer: https://ediri.eu.auth0.com/
clientID: %s
clientSecret: %s
requestedIDTokenClaims:
 groups:
  essential: true
requestedScopes:
- openid
- profile
- email
- 'https://example.com/claims/groups'`, clientId, clientSecret), nil
		})

	argocdReleaseName := "argo"

	argocdNS, err := v1.NewNamespace(ctx, "argo-ns", &v1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("argo"),
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	argocd, err := helm.NewRelease(ctx, "argocd-helm", &helm.ReleaseArgs{
		Name:      pulumi.String(argocdReleaseName),
		Chart:     pulumi.String("argo-cd"),
		Version:   pulumi.String("3.29.4"),
		Namespace: argocdNS.Metadata.Name(),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://argoproj.github.io/argo-helm"),
		},
		Values: pulumi.Map{
			"controller": pulumi.Map{
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
				"serviceAccount": pulumi.Map{
					"automountServiceAccountToken": pulumi.Bool(true),
				},
				"metrics": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"serviceMonitor": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
			},
			"dex": pulumi.Map{
				"enabled":      pulumi.Bool(false),
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
				"serviceAccount": pulumi.Map{
					"automountServiceAccountToken": pulumi.Bool(true),
				},
			},
			"redis": pulumi.Map{
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
				"serviceAccount": pulumi.Map{
					"automountServiceAccountToken": pulumi.Bool(true),
				},
				"metrics": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"serviceMonitor": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
			},
			"server": pulumi.Map{
				"metrics": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"serviceMonitor": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
				"config": pulumi.Map{
					"url":         pulumi.String("https://argocd.ediri.online"),
					"oidc.config": oidcString,
				},
				"rbacConfig": pulumi.Map{
					"policy.default": pulumi.String("role:readonly"),
					"policy.csv": pulumi.String(`g, argo-admins, role:admin
`),
					"scopes": pulumi.String("[https://example.com/claims/groups, email]"),
				},
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
				"extraArgs": pulumi.Array{
					pulumi.String("--insecure"),
				},
				"certificate": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"domain":  pulumi.String("argocd.ediri.online"),
					"issuer": pulumi.Map{
						"kind": pulumi.String("ClusterIssuer"),
						"name": pulumi.String("letsencrypt-staging"),
					},
					"secretName": pulumi.String("ediri-online-tls"),
				},
				"ingress": pulumi.Map{
					"ingressClassName": pulumi.String("nginx"),
					"hosts": pulumi.Array{
						pulumi.String("argocd.ediri.online"),
					},
					"enabled": pulumi.Bool(true),
					"tls": pulumi.Array{
						pulumi.Map{
							"hosts": pulumi.Array{
								pulumi.String("argocd.ediri.online"),
							},
							"secretName": pulumi.String("ediri-online-tls"),
						},
					},
					"annotations": pulumi.Map{
						"external-dns.alpha.kubernetes.io/hostname": pulumi.String("argocd.ediri.online"),
						"external-dns.alpha.kubernetes.io/ttl":      pulumi.String("60"),
					},
				},
			},
			"repoServer": pulumi.Map{
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
				"serviceAccount": pulumi.Map{
					"automountServiceAccountToken": pulumi.Bool(true),
				},
				"metrics": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"serviceMonitor": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(argocdNS))
	if err != nil {
		return nil, err
	}

	_, err = v1.NewConfigMap(ctx, "argocd-dashboard", &v1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("argocd-dashboard"),
			Namespace: argocdNS.Metadata.Name(),
			Labels: pulumi.StringMap{
				"grafana_dashboard": pulumi.String("1"),
			},
		},
		Data: pulumi.StringMap{
			"argocd-dashboard.json": pulumi.Sprintf(`{
  "__inputs": [],
  "__requires": [
    {
      "type": "panel",
      "id": "gauge",
      "name": "Gauge",
      "version": ""
    },
    {
      "type": "grafana",
      "id": "grafana",
      "name": "Grafana",
      "version": "8.0.0"
    },
    {
      "type": "panel",
      "id": "graph",
      "name": "Graph (old)",
      "version": ""
    },
    {
      "type": "panel",
      "id": "heatmap",
      "name": "Heatmap",
      "version": ""
    },
    {
      "type": "datasource",
      "id": "prometheus",
      "name": "Prometheus",
      "version": "1.0.0"
    },
    {
      "type": "panel",
      "id": "stat",
      "name": "Stat",
      "version": ""
    },
    {
      "type": "panel",
      "id": "text",
      "name": "Text",
      "version": ""
    }
  ],
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": "-- Grafana --",
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "gnetId": 14584,
  "graphTooltip": 0,
  "id": null,
  "iteration": 1623677737451,
  "links": [],
  "panels": [
    {
      "collapsed": false,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 0
      },
      "id": 68,
      "panels": [],
      "title": "Overview",
      "type": "row"
    },
    {
      "datasource": "$datasource",
      "gridPos": {
        "h": 4,
        "w": 2,
        "x": 0,
        "y": 1
      },
      "id": 26,
      "links": [],
      "options": {
        "content": "![argoimage](https://avatars1.githubusercontent.com/u/30269780?s=110&v=4)",
        "mode": "markdown"
      },
      "pluginVersion": "8.0.0",
      "transparent": true,
      "type": "text"
    },
    {
      "cacheTimeout": null,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          },
          "unit": "dtdurations"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 4,
        "w": 3,
        "x": 2,
        "y": 1
      },
      "id": 32,
      "interval": null,
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "value",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "lastNotNull"
          ],
          "fields": "",
          "values": false
        },
        "text": {},
        "textMode": "auto"
      },
      "pluginVersion": "8.0.0",
      "targets": [
        {
          "expr": "time() - max(process_start_time_seconds{job=\"%[1]s-argocd-server-metrics\",namespace=~\"$namespace\"})",
          "format": "time_series",
          "intervalFactor": 1,
          "refId": "A"
        }
      ],
      "title": "Uptime",
      "type": "stat"
    },
    {
      "cacheTimeout": null,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          },
          "unit": "none"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 4,
        "w": 3,
        "x": 5,
        "y": 1
      },
      "id": 94,
      "interval": null,
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "value",
        "graphMode": "area",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "lastNotNull"
          ],
          "fields": "",
          "values": false
        },
        "text": {},
        "textMode": "auto"
      },
      "pluginVersion": "8.0.0",
      "targets": [
        {
          "expr": "count(count by (server) (argocd_cluster_info{namespace=~\"$namespace\"}))",
          "format": "time_series",
          "instant": false,
          "intervalFactor": 1,
          "refId": "A"
        }
      ],
      "timeFrom": null,
      "timeShift": null,
      "title": "Clusters",
      "type": "stat"
    },
    {
      "cacheTimeout": null,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          },
          "unit": "none"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 4,
        "w": 3,
        "x": 8,
        "y": 1
      },
      "id": 75,
      "interval": null,
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "value",
        "graphMode": "area",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "lastNotNull"
          ],
          "fields": "",
          "values": false
        },
        "text": {},
        "textMode": "auto"
      },
      "pluginVersion": "8.0.0",
      "repeat": null,
      "repeatDirection": "h",
      "targets": [
        {
          "expr": "sum(argocd_app_info{namespace=~\"$namespace\",dest_server=~\"$cluster\",health_status=~\"$health_status\",sync_status=~\"$sync_status\"})",
          "format": "time_series",
          "instant": false,
          "intervalFactor": 1,
          "refId": "A"
        }
      ],
      "timeFrom": null,
      "timeShift": null,
      "title": "Applications",
      "type": "stat"
    },
    {
      "cacheTimeout": null,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          },
          "unit": "none"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 4,
        "w": 3,
        "x": 11,
        "y": 1
      },
      "id": 107,
      "interval": null,
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "value",
        "graphMode": "area",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "lastNotNull"
          ],
          "fields": "",
          "values": false
        },
        "text": {},
        "textMode": "auto"
      },
      "pluginVersion": "8.0.0",
      "targets": [
        {
          "expr": "count(count by (repo) (argocd_app_info{namespace=~\"$namespace\"}))",
          "format": "time_series",
          "instant": false,
          "intervalFactor": 1,
          "refId": "A"
        }
      ],
      "timeFrom": null,
      "timeShift": null,
      "title": "Repositories",
      "type": "stat"
    },
    {
      "cacheTimeout": null,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "mappings": [
            {
              "id": 0,
              "op": "=",
              "text": "0",
              "type": 1,
              "value": "null"
            }
          ],
          "max": 100,
          "min": 0,
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          },
          "unit": "none"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 4,
        "w": 3,
        "x": 14,
        "y": 1
      },
      "id": 100,
      "links": [],
      "options": {
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "lastNotNull"
          ],
          "fields": "",
          "values": false
        },
        "showThresholdLabels": false,
        "showThresholdMarkers": true,
        "text": {}
      },
      "pluginVersion": "8.0.0",
      "repeatDirection": "h",
      "targets": [
        {
          "expr": "sum(argocd_app_info{namespace=~\"$namespace\",dest_server=~\"$cluster\",operation!=\"\"})",
          "format": "time_series",
          "instant": true,
          "intervalFactor": 1,
          "legendFormat": "",
          "refId": "A"
        }
      ],
      "timeFrom": null,
      "timeShift": null,
      "title": "Operations",
      "type": "gauge"
    },
    {
      "aliasColors": {},
      "bars": false,
      "cacheTimeout": null,
      "dashLength": 10,
      "dashes": false,
      "datasource": "$datasource",
      "decimals": null,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "gridPos": {
        "h": 4,
        "w": 7,
        "x": 17,
        "y": 1
      },
      "hiddenSeries": false,
      "id": 28,
      "legend": {
        "alignAsTable": true,
        "avg": false,
        "current": true,
        "hideEmpty": true,
        "hideZero": true,
        "max": false,
        "min": false,
        "rightSide": true,
        "show": true,
        "sort": "current",
        "sortDesc": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 1,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "paceLength": 10,
      "percentage": false,
      "pluginVersion": "8.0.0",
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": true,
      "steppedLine": false,
      "targets": [
        {
          "expr": "sum(argocd_app_info{namespace=~\"$namespace\",dest_server=~\"$cluster\",health_status=~\"$health_status\",sync_status=~\"$sync_status\"}) by (namespace)",
          "format": "time_series",
          "instant": false,
          "intervalFactor": 1,
          "legendFormat": "{{namespace}}",
          "refId": "A"
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Applications",
      "tooltip": {
        "shared": false,
        "sort": 2,
        "value_type": "individual"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "decimals": 0,
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        },
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "collapsed": true,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 5
      },
      "id": 77,
      "panels": [
        {
          "aliasColors": {
            "Degraded": "semi-dark-red",
            "Healthy": "green",
            "Missing": "semi-dark-purple",
            "Progressing": "semi-dark-blue",
            "Suspended": "semi-dark-orange",
            "Unknown": "rgb(255, 255, 255)"
          },
          "bars": false,
          "cacheTimeout": null,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "decimals": null,
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 6,
            "w": 12,
            "x": 0,
            "y": 6
          },
          "hiddenSeries": false,
          "id": 105,
          "interval": "",
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": false,
            "hideZero": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": null,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(argocd_app_info{namespace=~\"$namespace\",dest_server=~\"$cluster\",health_status=~\"$health_status\",sync_status=~\"$sync_status\",health_status!=\"\"}) by (health_status)",
              "format": "time_series",
              "instant": false,
              "intervalFactor": 1,
              "legendFormat": "{{health_status}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Health Status",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 2,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {
            "Degraded": "semi-dark-red",
            "Healthy": "green",
            "Missing": "semi-dark-purple",
            "OutOfSync": "semi-dark-yellow",
            "Progressing": "semi-dark-blue",
            "Suspended": "semi-dark-orange",
            "Synced": "semi-dark-green",
            "Unknown": "rgb(255, 255, 255)"
          },
          "bars": false,
          "cacheTimeout": null,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "decimals": null,
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 6,
            "w": 12,
            "x": 12,
            "y": 6
          },
          "hiddenSeries": false,
          "id": 106,
          "interval": "",
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": false,
            "hideZero": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": null,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(argocd_app_info{namespace=~\"$namespace\",dest_server=~\"$cluster\",health_status=~\"$health_status\",sync_status=~\"$sync_status\",health_status!=\"\"}) by (sync_status)",
              "format": "time_series",
              "instant": false,
              "intervalFactor": 1,
              "legendFormat": "{{sync_status}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Sync Status",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 2,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Application Status",
      "type": "row"
    },
    {
      "collapsed": true,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 6
      },
      "id": 104,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "decimals": null,
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 6,
            "w": 24,
            "x": 0,
            "y": 3
          },
          "hiddenSeries": false,
          "id": 56,
          "interval": "",
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "total",
            "sortDesc": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 1,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(round(increase(argocd_app_sync_total{namespace=~\"$namespace\",dest_server=~\"$cluster\"}[$interval]))) by ($grouping)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{$grouping}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Sync Activity",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "decimals": 0,
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "decimals": -12,
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "decimals": null,
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 5,
            "w": 24,
            "x": 0,
            "y": 9
          },
          "hiddenSeries": false,
          "id": 73,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "hideEmpty": true,
            "hideZero": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "total",
            "sortDesc": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(round(increase(argocd_app_sync_total{namespace=~\"$namespace\",phase=~\"Error|Failed\",dest_server=~\"$cluster\"}[$interval]))) by ($grouping, phase)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{phase}}: {{$grouping}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Sync Failures",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "decimals": 0,
              "format": "none",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Sync Stats",
      "type": "row"
    },
    {
      "collapsed": true,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 7
      },
      "id": 64,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 6,
            "w": 24,
            "x": 0,
            "y": 12
          },
          "hiddenSeries": false,
          "id": 58,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "total",
            "sortDesc": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(argocd_app_reconcile_count{namespace=~\"$namespace\",dest_server=~\"$cluster\"}[$interval])) by ($grouping)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{$grouping}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Reconciliation Activity",
          "tooltip": {
            "shared": false,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "cards": {
            "cardPadding": null,
            "cardRound": null
          },
          "color": {
            "cardColor": "#b4ff00",
            "colorScale": "sqrt",
            "colorScheme": "interpolateSpectral",
            "exponent": 0.5,
            "min": null,
            "mode": "spectrum"
          },
          "dataFormat": "tsbuckets",
          "datasource": "$datasource",
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 18
          },
          "heatmap": {},
          "hideZeroBuckets": false,
          "highlightCards": true,
          "id": 60,
          "legend": {
            "show": true
          },
          "links": [],
          "options": {},
          "reverseYBuckets": false,
          "targets": [
            {
              "expr": "sum(increase(argocd_app_reconcile_bucket{namespace=~\"$namespace\"}[$interval])) by (le)",
              "format": "heatmap",
              "instant": false,
              "intervalFactor": 10,
              "legendFormat": "{{le}}",
              "refId": "A"
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Reconciliation Performance",
          "tooltip": {
            "show": true,
            "showHistogram": true
          },
          "tooltipDecimals": 0,
          "type": "heatmap",
          "xAxis": {
            "show": true
          },
          "xBucketNumber": null,
          "xBucketSize": null,
          "yAxis": {
            "decimals": null,
            "format": "short",
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true,
            "splitFactor": null
          },
          "yBucketBound": "auto",
          "yBucketNumber": null,
          "yBucketSize": null
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 6,
            "w": 24,
            "x": 0,
            "y": 25
          },
          "hiddenSeries": false,
          "id": 80,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": null,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(argocd_app_k8s_request_total{namespace=~\"$namespace\",server=~\"$cluster\"}[$interval])) by (verb, resource_kind)",
              "format": "time_series",
              "instant": false,
              "intervalFactor": 1,
              "legendFormat": "{{verb}} {{resource_kind}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "K8s API Activity",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 31
          },
          "hiddenSeries": false,
          "id": 96,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": false,
            "show": true,
            "sideWidth": null,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(workqueue_depth{namespace=~\"$namespace\",name=~\"app_.*\"}) by (name)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{name}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Workqueue Depth",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "decimals": null,
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 31
          },
          "hiddenSeries": false,
          "id": 98,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(argocd_kubectl_exec_pending{namespace=~\"$namespace\"}) by (command)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{command}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Pending kubectl run",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "decimals": 0,
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "decimals": 0,
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Controller Stats",
      "type": "row"
    },
    {
      "collapsed": true,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 8
      },
      "id": 102,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 26
          },
          "hiddenSeries": false,
          "id": 34,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "connected",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "go_memstats_heap_alloc_bytes{job=\"%[1]s-argocd-application-controller-metrics\",namespace=~\"$namespace\"}",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{namespace}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Memory Usage",
          "tooltip": {
            "shared": false,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 33
          },
          "hiddenSeries": false,
          "id": 108,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "avg",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "connected",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "irate(process_cpu_seconds_total{job=\"%[1]s-argocd-application-controller-metrics\",namespace=~\"$namespace\"}[1m])",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{namespace}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "CPU Usage",
          "tooltip": {
            "shared": false,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "decimals": 1,
              "format": "none",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 40
          },
          "hiddenSeries": false,
          "id": 62,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": false,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "go_goroutines{job=\"%[1]s-argocd-application-controller-metrics\",namespace=~\"$namespace\"}",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{namespace}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Goroutines",
          "tooltip": {
            "shared": false,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Controller Telemetry",
      "type": "row"
    },
    {
      "collapsed": true,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 9
      },
      "id": 88,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 27
          },
          "hiddenSeries": false,
          "id": 90,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(argocd_cluster_api_resource_objects{namespace=~\"$namespace\",server=~\"$cluster\"}) by (server)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{server}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Resource Objects Count",
          "tooltip": {
            "shared": false,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 6,
            "w": 24,
            "x": 0,
            "y": 34
          },
          "hiddenSeries": false,
          "id": 92,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "  sum(argocd_cluster_api_resources{namespace=~\"$namespace\",server=~\"$cluster\"}) by (server)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{server}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "API Resources Count",
          "tooltip": {
            "shared": false,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 40
          },
          "hiddenSeries": false,
          "id": 86,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(argocd_cluster_events_total{namespace=~\"$namespace\",server=~\"$cluster\"}[$interval])) by (server)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{server}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Cluster Events Count",
          "tooltip": {
            "shared": false,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Cluster Stats",
      "type": "row"
    },
    {
      "collapsed": true,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 10
      },
      "id": 70,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 7
          },
          "hiddenSeries": false,
          "id": 82,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": false,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(argocd_git_request_total{request_type=\"ls-remote\", namespace=~\"$namespace\"}[10m])) by (namespace)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{namespace}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Git Requests (ls-remote)",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 7
          },
          "hiddenSeries": false,
          "id": 84,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": false,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(argocd_git_request_total{request_type=\"fetch\", namespace=~\"$namespace\"}[10m])) by (namespace)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{namespace}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Git Requests (checkout)",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "cards": {
            "cardPadding": null,
            "cardRound": null
          },
          "color": {
            "cardColor": "#b4ff00",
            "colorScale": "sqrt",
            "colorScheme": "interpolateSpectral",
            "exponent": 0.5,
            "mode": "spectrum"
          },
          "dataFormat": "tsbuckets",
          "datasource": "$datasource",
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 15
          },
          "heatmap": {},
          "hideZeroBuckets": false,
          "highlightCards": true,
          "id": 114,
          "legend": {
            "show": false
          },
          "options": {},
          "reverseYBuckets": false,
          "targets": [
            {
              "expr": "sum(increase(argocd_git_request_duration_seconds_bucket{request_type=\"fetch\", namespace=~\"$namespace\"}[$interval])) by (le)",
              "format": "heatmap",
              "intervalFactor": 10,
              "legendFormat": "{{le}}",
              "refId": "A"
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Git Fetch Performance",
          "tooltip": {
            "show": true,
            "showHistogram": false
          },
          "type": "heatmap",
          "xAxis": {
            "show": true
          },
          "xBucketNumber": null,
          "xBucketSize": null,
          "yAxis": {
            "decimals": null,
            "format": "short",
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true,
            "splitFactor": null
          },
          "yBucketBound": "auto",
          "yBucketNumber": null,
          "yBucketSize": null
        },
        {
          "cards": {
            "cardPadding": null,
            "cardRound": null
          },
          "color": {
            "cardColor": "#b4ff00",
            "colorScale": "sqrt",
            "colorScheme": "interpolateSpectral",
            "exponent": 0.5,
            "mode": "spectrum"
          },
          "dataFormat": "tsbuckets",
          "datasource": "$datasource",
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 15
          },
          "heatmap": {},
          "hideZeroBuckets": false,
          "highlightCards": true,
          "id": 116,
          "legend": {
            "show": false
          },
          "options": {},
          "reverseYBuckets": false,
          "targets": [
            {
              "expr": "sum(increase(argocd_git_request_duration_seconds_bucket{request_type=\"ls-remote\", namespace=~\"$namespace\"}[$interval])) by (le)",
              "format": "heatmap",
              "intervalFactor": 10,
              "legendFormat": "{{le}}",
              "refId": "A"
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Git Ls-Remote Performance",
          "tooltip": {
            "show": true,
            "showHistogram": false
          },
          "type": "heatmap",
          "xAxis": {
            "show": true
          },
          "xBucketNumber": null,
          "xBucketSize": null,
          "yAxis": {
            "decimals": null,
            "format": "short",
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true,
            "splitFactor": null
          },
          "yBucketBound": "auto",
          "yBucketNumber": null,
          "yBucketSize": null
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 8,
            "w": 24,
            "x": 0,
            "y": 23
          },
          "hiddenSeries": false,
          "id": 71,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": false,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "connected",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "go_memstats_heap_alloc_bytes{job=\"%[1]s-argocd-repo-server-metrics\",namespace=~\"$namespace\"}",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{pod}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Memory Used",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 31
          },
          "hiddenSeries": false,
          "id": 72,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": false,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "go_goroutines{job=\"%[1]s-argocd-repo-server-metrics\",namespace=~\"$namespace\"}",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{pod}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Goroutines",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Repo Server Stats",
      "type": "row"
    },
    {
      "collapsed": true,
      "datasource": "$datasource",
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 11
      },
      "id": 66,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "gridPos": {
            "h": 8,
            "w": 24,
            "x": 0,
            "y": 89
          },
          "id": 61,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": false,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "connected",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "go_memstats_heap_alloc_bytes{job=\"%[1]s-argocd-server-metrics\",namespace=~\"$namespace\"}",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{pod}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Memory Used",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 24,
            "x": 0,
            "y": 97
          },
          "id": 36,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": false,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "go_goroutines{job=\"%[1]s-argocd-server-metrics\",namespace=~\"$namespace\"}",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{pod}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Goroutines",
          "tooltip": {
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 24,
            "x": 0,
            "y": 106
          },
          "id": 38,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "connected",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "go_gc_duration_seconds{job=\"%[1]s-argocd-server-metrics\", quantile=\"1\", namespace=~\"$namespace\"}",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{pod}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "GC Time Quantiles",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "content": "#### gRPC Services:",
          "gridPos": {
            "h": 2,
            "w": 24,
            "x": 0,
            "y": 115
          },
          "id": 54,
          "links": [],
          "mode": "markdown",
          "title": "",
          "transparent": true,
          "type": "text"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "decimals": null,
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 12,
            "x": 0,
            "y": 117
          },
          "id": 40,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "rightSide": false,
            "show": true,
            "sort": "total",
            "sortDesc": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(grpc_server_handled_total{job=\"%[1]s-argocd-server-metrics\",grpc_service=\"application.ApplicationService\",namespace=~\"$namespace\"}[$interval])) by (grpc_code, grpc_method)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{grpc_code}},{{grpc_method}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "ApplicationService Requests",
          "tooltip": {
            "shared": false,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 12,
            "x": 12,
            "y": 117
          },
          "id": 42,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "rightSide": false,
            "show": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(grpc_server_handled_total{job=\"%[1]s-argocd-server-metrics\",grpc_service=\"cluster.ClusterService\",namespace=~\"$namespace\"}[$interval])) by (grpc_code, grpc_method)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{grpc_code}},{{grpc_method}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "ClusterService Requests",
          "tooltip": {
            "shared": false,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 12,
            "x": 0,
            "y": 126
          },
          "id": 44,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "rightSide": false,
            "show": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(grpc_server_handled_total{job=\"%[1]s-argocd-server-metrics\",grpc_service=\"project.ProjectService\",namespace=~\"$namespace\"}[$interval])) by (grpc_code, grpc_method)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{grpc_code}},{{grpc_method}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "ProjectService Requests",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 12,
            "x": 12,
            "y": 126
          },
          "id": 46,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "show": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(grpc_server_handled_total{job=\"%[1]s-argocd-server-metrics\",grpc_service=\"repository.RepositoryService\",namespace=~\"$namespace\"}[$interval])) by (grpc_code, grpc_method)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{grpc_code}},{{grpc_method}}",
              "refId": "A"
            }
          ],
          "thresholds": [
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "yaxis": "left"
            }
          ],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "RepositoryService Requests",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 12,
            "x": 0,
            "y": 135
          },
          "id": 48,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "show": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(grpc_server_handled_total{job=\"%[1]s-argocd-server-metrics\",grpc_service=\"session.SessionService\",namespace=~\"$namespace\"}[$interval])) by (grpc_code, grpc_method)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{grpc_code}},{{grpc_method}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "SessionService Requests",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 12,
            "x": 12,
            "y": 135
          },
          "id": 49,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "show": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(grpc_server_handled_total{job=\"%[1]s-argocd-server-metrics\",grpc_service=\"version.VersionService\",namespace=~\"$namespace\"}[$interval])) by (grpc_code, grpc_method)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{grpc_code}},{{grpc_method}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "VersionService Requests",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 12,
            "x": 0,
            "y": 144
          },
          "id": 50,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "show": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(grpc_server_handled_total{job=\"%[1]s-argocd-server-metrics\",grpc_service=\"account.AccountService\",namespace=~\"$namespace\"}[$interval])) by (grpc_code, grpc_method)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{grpc_code}},{{grpc_method}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "AccountService Requests",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "$datasource",
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 12,
            "x": 12,
            "y": 144
          },
          "id": 99,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "show": true,
            "total": true,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "paceLength": 10,
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(grpc_server_handled_total{job=\"%[1]s-argocd-server-metrics\",grpc_service=\"settings.SettingsService\",namespace=~\"$namespace\"}[$interval])) by (grpc_code, grpc_method)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{grpc_code}},{{grpc_method}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "SettingsService Requests",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Server Stats",
      "type": "row"
    },
    {
      "collapsed": true,
      "datasource": null,
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 12
      },
      "id": 110,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": null,
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 9
          },
          "hiddenSeries": false,
          "id": 112,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "nullPointMode": "null",
          "options": {
            "dataLinks": []
          },
          "percentage": false,
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(argocd_redis_request_total{namespace=~\"$namespace\"}[$interval])) by (failed)",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Requests by result",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Redis Stats",
      "type": "row"
    }
  ],
  "refresh": false,
  "schemaVersion": 30,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": [
      {
        "current": {
          "selected": false,
          "text": "Prometheus",
          "value": "Prometheus"
        },
        "description": null,
        "error": null,
        "hide": 0,
        "includeAll": false,
        "label": null,
        "multi": false,
        "name": "datasource",
        "options": [],
        "query": "prometheus",
        "refresh": 1,
        "regex": "",
        "skipUrlSync": false,
        "type": "datasource"
      },
      {
        "allValue": ".*",
        "current": {},
        "datasource": "$datasource",
        "definition": "label_values(kube_pod_info, namespace)",
        "description": null,
        "error": null,
        "hide": 0,
        "includeAll": true,
        "label": null,
        "multi": false,
        "name": "namespace",
        "options": [],
        "query": {
          "query": "label_values(kube_pod_info, namespace)",
          "refId": "Prometheus-namespace-Variable-Query"
        },
        "refresh": 1,
        "regex": ".*argocd.*",
        "skipUrlSync": false,
        "sort": 0,
        "tagValuesQuery": "",
        "tagsQuery": "",
        "type": "query",
        "useTags": false
      },
      {
        "auto": true,
        "auto_count": 30,
        "auto_min": "1m",
        "current": {
          "selected": false,
          "text": "auto",
          "value": "$__auto_interval_interval"
        },
        "description": null,
        "error": null,
        "hide": 0,
        "label": null,
        "name": "interval",
        "options": [
          {
            "selected": true,
            "text": "auto",
            "value": "$__auto_interval_interval"
          },
          {
            "selected": false,
            "text": "1m",
            "value": "1m"
          },
          {
            "selected": false,
            "text": "5m",
            "value": "5m"
          },
          {
            "selected": false,
            "text": "10m",
            "value": "10m"
          },
          {
            "selected": false,
            "text": "30m",
            "value": "30m"
          },
          {
            "selected": false,
            "text": "1h",
            "value": "1h"
          },
          {
            "selected": false,
            "text": "2h",
            "value": "2h"
          },
          {
            "selected": false,
            "text": "4h",
            "value": "4h"
          },
          {
            "selected": false,
            "text": "8h",
            "value": "8h"
          }
        ],
        "query": "1m,5m,10m,30m,1h,2h,4h,8h",
        "refresh": 2,
        "skipUrlSync": false,
        "type": "interval"
      },
      {
        "allValue": "",
        "current": {
          "selected": true,
          "text": "namespace",
          "value": "namespace"
        },
        "description": null,
        "error": null,
        "hide": 0,
        "includeAll": false,
        "label": null,
        "multi": false,
        "name": "grouping",
        "options": [
          {
            "selected": true,
            "text": "namespace",
            "value": "namespace"
          },
          {
            "selected": false,
            "text": "name",
            "value": "name"
          },
          {
            "selected": false,
            "text": "project",
            "value": "project"
          }
        ],
        "query": "namespace,name,project",
        "skipUrlSync": false,
        "type": "custom"
      },
      {
        "allValue": ".*",
        "current": {},
        "datasource": "$datasource",
        "definition": "label_values(argocd_cluster_info, server)",
        "description": null,
        "error": null,
        "hide": 0,
        "includeAll": true,
        "label": null,
        "multi": false,
        "name": "cluster",
        "options": [],
        "query": {
          "query": "label_values(argocd_cluster_info, server)",
          "refId": "Prometheus-cluster-Variable-Query"
        },
        "refresh": 1,
        "regex": "",
        "skipUrlSync": false,
        "sort": 1,
        "tagValuesQuery": "",
        "tagsQuery": "",
        "type": "query",
        "useTags": false
      },
      {
        "allValue": ".*",
        "current": {
          "selected": true,
          "text": "All",
          "value": "$__all"
        },
        "description": null,
        "error": null,
        "hide": 0,
        "includeAll": true,
        "label": null,
        "multi": false,
        "name": "health_status",
        "options": [
          {
            "selected": true,
            "text": "All",
            "value": "$__all"
          },
          {
            "selected": false,
            "text": "Healthy",
            "value": "Healthy"
          },
          {
            "selected": false,
            "text": "Progressing",
            "value": "Progressing"
          },
          {
            "selected": false,
            "text": "Suspended",
            "value": "Suspended"
          },
          {
            "selected": false,
            "text": "Missing",
            "value": "Missing"
          },
          {
            "selected": false,
            "text": "Degraded",
            "value": "Degraded"
          },
          {
            "selected": false,
            "text": "Unknown",
            "value": "Unknown"
          }
        ],
        "query": "Healthy,Progressing,Suspended,Missing,Degraded,Unknown",
        "skipUrlSync": false,
        "type": "custom"
      },
      {
        "allValue": ".*",
        "current": {
          "selected": true,
          "text": "All",
          "value": "$__all"
        },
        "description": null,
        "error": null,
        "hide": 0,
        "includeAll": true,
        "label": null,
        "multi": false,
        "name": "sync_status",
        "options": [
          {
            "selected": true,
            "text": "All",
            "value": "$__all"
          },
          {
            "selected": false,
            "text": "Synced",
            "value": "Synced"
          },
          {
            "selected": false,
            "text": "OutOfSync",
            "value": "OutOfSync"
          },
          {
            "selected": false,
            "text": "Unknown",
            "value": "Unknown"
          }
        ],
        "query": "Synced,OutOfSync,Unknown",
        "skipUrlSync": false,
        "type": "custom"
      }
    ]
  },
  "time": {
    "from": "now-30m",
    "to": "now"
  },
  "timepicker": {
    "refresh_intervals": [
      "5s",
      "10s",
      "30s",
      "1m",
      "5m",
      "15m",
      "30m",
      "1h",
      "2h",
      "1d"
    ],
    "time_options": [
      "5m",
      "15m",
      "1h",
      "6h",
      "12h",
      "24h",
      "2d",
      "7d",
      "30d"
    ]
  },
  "timezone": "",
  "title": "ArgoCD",
  "uid": "qPkgGHg7k",
  "version": 1,
  "description": "Official ArgoCD Dashboard as of June 2021"
}`, argocdReleaseName),
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(argocdNS))
	if err != nil {
		return nil, err
	}

	return argocd, nil
}
