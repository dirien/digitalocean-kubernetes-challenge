package charts

import (
	"fmt"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func createClaim(ctx *pulumi.Context, args CreateChartArgs, name string, monitoring *v1.Namespace, storage string, parent *helm.Release) (*v1.PersistentVolumeClaim, error) {
	pvc, err := v1.NewPersistentVolumeClaim(ctx, name, &v1.PersistentVolumeClaimArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(fmt.Sprintf("%s-pvc", name)),
			Namespace: monitoring.Metadata.Name(),
		},
		Spec: &v1.PersistentVolumeClaimSpecArgs{
			AccessModes: pulumi.StringArray{
				pulumi.String("ReadWriteOnce"),
			},
			Resources: &v1.ResourceRequirementsArgs{
				Requests: pulumi.StringMap{
					"storage": pulumi.String(storage),
				},
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(parent), pulumi.Parent(monitoring))
	if err != nil {
		return nil, err
	}
	return pvc, nil
}

func NewMonitoring(ctx *pulumi.Context, args CreateChartArgs, parent *helm.Release) (*helm.Release, error) {

	_, err := helm.NewRelease(ctx, "metrics-server", &helm.ReleaseArgs{
		Name:        pulumi.String("metrics-server"),
		Chart:       pulumi.String("metrics-server"),
		Version:     pulumi.String("3.7.0"),
		Namespace:   pulumi.String("kube-system"),
		ForceUpdate: pulumi.Bool(true),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://kubernetes-sigs.github.io/metrics-server"),
		},
		Values: pulumi.Map{
			"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
			"tolerations":  GetPlacement(args.Cloud)["tolerations"],
		},
	}, pulumi.Provider(args.Provider))
	if err != nil {
		return nil, err
	}

	monitoring, err := v1.NewNamespace(ctx, "monitoring", &v1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("monitoring"),
		},
	}, pulumi.Provider(args.Provider))
	if err != nil {
		return nil, err
	}

	grafanaClaim, err := createClaim(ctx, args, "grafana", monitoring, "10Gi", parent)
	if err != nil {
		return nil, err
	}

	_, err = apiextensions.NewCustomResource(ctx, "grafana-certificate", &apiextensions.CustomResourceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("grafana-certificate"),
			Namespace: monitoring.Metadata.Name(),
		},
		ApiVersion: pulumi.String("cert-manager.io/v1"),
		Kind:       pulumi.String("Certificate"),
		OtherFields: kubernetes.UntypedArgs{
			"spec": &pulumi.Map{
				"commonName": pulumi.String("grafana.ediri.online"),
				"dnsNames": pulumi.StringArray{
					pulumi.String("grafana.ediri.online"),
				},
				"issuerRef": &pulumi.Map{
					"name": pulumi.String("letsencrypt-staging"),
					"kind": pulumi.String("ClusterIssuer"),
				},
				"secretName": pulumi.String("grafana-tls"),
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(parent), pulumi.Parent(monitoring))
	if err != nil {
		return nil, err
	}

	loki, err := helm.NewRelease(ctx, "grafana-loki", &helm.ReleaseArgs{
		Name:            pulumi.String("grafana-loki"),
		Chart:           pulumi.String("loki-stack"),
		Version:         pulumi.String("2.5.0"),
		Namespace:       monitoring.Metadata.Name(),
		CreateNamespace: pulumi.Bool(false),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://grafana.github.io/helm-charts"),
		},
		Values: pulumi.Map{
			"promtail": pulumi.Map{
				"serviceMonitor": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
			},
			"loki": pulumi.Map{
				"serviceMonitor": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
				"persistence": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"size":    pulumi.String("50Gi"),
				},
				"config": pulumi.Map{
					"chunk_store_config": pulumi.Map{
						"max_look_back_period": pulumi.String("0s"),
					},
					"table_manager": pulumi.Map{
						"retention_deletes_enabled": pulumi.Bool(false),
						"retention_period":          pulumi.String("0s"),
					},
					"storage_config": pulumi.Map{
						"aws": pulumi.Map{
							"bucketnames":       args.Cloud.GetStringOutput(pulumi.String("loki.bucket.Name")),
							"endpoint":          pulumi.Sprintf("%s.digitaloceanspaces.com", args.Cloud.GetStringOutput(pulumi.String("loki.bucket.region"))),
							"region":            args.Cloud.GetStringOutput(pulumi.String("loki.bucket.region")),
							"access_key_id":     pulumi.String(config.Get(ctx, "spaces_access_id")),
							"secret_access_key": pulumi.String(config.Get(ctx, "spaces_secret_key")),
							"s3forcepathstyle":  pulumi.Bool(true),
						},
						"boltdb_shipper": pulumi.Map{
							"active_index_directory": pulumi.String("/data/loki/index"),
							"cache_location":         pulumi.String("/data/loki/index_cache"),
							"cache_ttl":              pulumi.String("24h"),
							"shared_store":           pulumi.String("aws"),
						},
					},
					"schema_config": pulumi.Map{
						"configs": pulumi.Array{
							pulumi.Map{
								"from":         pulumi.String("2021-01-01"),
								"store":        pulumi.String("boltdb-shipper"),
								"object_store": pulumi.String("aws"),
								"schema":       pulumi.String("v11"),
								"index": pulumi.Map{
									"prefix": pulumi.String("index_"),
									"period": pulumi.String("24h"),
								},
							},
						},
					},
				},
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(monitoring))
	if err != nil {
		return nil, err
	}

	lokiSvcUrl := pulumi.All(loki.Name).ApplyT(
		func(args []interface{}) string {
			svc := args[0].(*string)
			return fmt.Sprintf("http://%s.monitoring.svc.cluster.local:3100", *svc)
		})

	auth0Domain := config.Get(ctx, "auth0Domain")

	grafanaAdmin, err := v1.NewSecret(ctx, "grafana-admin-user", &v1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("grafana-admin-user"),
			Namespace: monitoring.Metadata.Name(),
		},
		StringData: pulumi.StringMap{
			"admin-user":     pulumi.String("admin"),
			"admin-password": pulumi.String(config.Get(ctx, "grafana-password")),
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(monitoring))
	if err != nil {
		return nil, err
	}

	stack, err := helm.NewRelease(ctx, "kube-prometheus-stack", &helm.ReleaseArgs{
		Name:            pulumi.String("kube-prometheus-stack"),
		Chart:           pulumi.String("kube-prometheus-stack"),
		Version:         pulumi.String("27.2.1"),
		Namespace:       monitoring.Metadata.Name(),
		CreateNamespace: pulumi.Bool(false),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://prometheus-community.github.io/helm-charts"),
		},
		Values: pulumi.Map{
			"prometheusOperator": &pulumi.Map{
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
			},
			"grafana": pulumi.Map{
				"enabled":      pulumi.Bool(true),
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
				"persistence": pulumi.Map{
					"enabled":       pulumi.Bool(true),
					"existingClaim": grafanaClaim.Metadata.Name(),
				},
				"sidecar": pulumi.Map{
					"dashboards": pulumi.Map{
						"searchNamespace": pulumi.String("ALL"),
					},
				},
				"admin": pulumi.Map{
					"existingSecret": grafanaAdmin.Metadata.Name(),
					"userKey":        pulumi.String("admin-user"),
					"passwordKey":    pulumi.String("admin-password"),
				},
				"additionalDataSources": pulumi.Array{
					pulumi.Map{
						"name": pulumi.String("Loki"),
						"type": pulumi.String("loki"),
						"url":  lokiSvcUrl,
						"jsonData": pulumi.Map{
							"maxLines": pulumi.Int(1000),
						},
					},
				},
				"grafana.ini": pulumi.Map{
					"paths": pulumi.Map{
						"data":         pulumi.String("/var/lib/grafana/"),
						"logs":         pulumi.String("/var/log/grafana"),
						"plugins":      pulumi.String("/var/lib/grafana/plugins"),
						"provisioning": pulumi.String("/etc/grafana/provisioning"),
					},
					"analytics": pulumi.Map{
						"check_for_updates": pulumi.Bool(true),
					},
					"log": pulumi.Map{
						"level": pulumi.String("debug"),
						"mode":  pulumi.String("console"),
					},
					"grafana_net": pulumi.Map{
						"url": pulumi.String("https://grafana.net"),
					},
					"server": pulumi.Map{
						"router_logging": pulumi.Bool(true),
						"root_url":       pulumi.String("https://grafana.ediri.online"),
					},
					"auth.basic": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
					"auth": pulumi.Map{
						"disable_login_form": pulumi.Bool(true),
					},
					"security": pulumi.Map{
						"disable_initial_admin_creation": pulumi.Bool(false),
					},
					"auth.generic_oauth": pulumi.Map{
						"enabled":               pulumi.Bool(true),
						"allow_sign_up":         pulumi.Bool(true),
						"allowed_organizations": pulumi.String(""),
						"name":                  pulumi.String("Auth0"),
						"client_id":             args.Auth.GetStringOutput(pulumi.String("grafana.clientId")),
						"client_secret":         args.Auth.GetStringOutput(pulumi.String("grafana.clientSecret")),
						"scopes":                pulumi.String("openid profile email"),
						"auth_url":              pulumi.String(fmt.Sprintf("https://%s/authorize", auth0Domain)),
						"token_url":             pulumi.String(fmt.Sprintf("https://%s/oauth/token", auth0Domain)),
						"api_url":               pulumi.String(fmt.Sprintf("https://%s/userinfo", auth0Domain)),
						"use_pkce":              pulumi.Bool(true),
						"role_attribute_path":   pulumi.String("contains(\"https://example.com/claims/groups\"[*], 'Admin') && 'Admin' || contains(\"https://example.com/claims/groups\"[*], 'Editor') && 'Editor' || 'Viewer'"),
					},
				},

				"ingress": pulumi.Map{
					"enabled":          pulumi.Bool(true),
					"ingressClassName": pulumi.String("nginx"),
					"annotations": pulumi.Map{
						"external-dns.alpha.kubernetes.io/hostname": pulumi.String("grafana.ediri.online"),
						"external-dns.alpha.kubernetes.io/ttl":      pulumi.String("60"),
					},
					"hosts": pulumi.StringArray{
						pulumi.String("grafana.ediri.online"),
					},
					"tls": pulumi.Array{
						pulumi.Map{
							"hosts": pulumi.StringArray{
								pulumi.String("grafana.ediri.online"),
							},
							"secretName": pulumi.String("grafana-tls"),
						},
					},
				},
			},
			"alertmanager": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"statefulSet": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
				"alertmanagerSpec": pulumi.Map{
					"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
					"tolerations":  GetPlacement(args.Cloud)["tolerations"],
					"storageSpec": pulumi.Map{
						"volumeClaimTemplate": pulumi.Map{
							"spec": pulumi.Map{
								"accessModes": pulumi.StringArray{
									pulumi.String("ReadWriteOnce"),
								},
								"resources": pulumi.Map{
									"requests": pulumi.Map{
										"storage": pulumi.String("50Gi"),
									},
								},
							},
						},
					},
				},
			},
			"prometheus": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"prometheusSpec": pulumi.Map{
					"serviceMonitorSelectorNilUsesHelmValues": pulumi.Bool(false),
					"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
					"tolerations":  GetPlacement(args.Cloud)["tolerations"],
					"storageSpec": pulumi.Map{
						"volumeClaimTemplate": pulumi.Map{
							"spec": pulumi.Map{
								"accessModes": pulumi.StringArray{
									pulumi.String("ReadWriteOnce"),
								},
								"resources": pulumi.Map{
									"requests": pulumi.Map{
										"storage": pulumi.String("50Gi"),
									},
								},
							},
						},
					},
				},
			},
			"kube-state-metrics": pulumi.Map{
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
			},
			"prometheus-node-exporter": pulumi.Map{
				"prometheus": pulumi.Map{
					"monitor": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(parent), pulumi.Parent(monitoring))
	if err != nil {
		return nil, err
	}

	return stack, nil
}
