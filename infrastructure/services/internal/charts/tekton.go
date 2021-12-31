package charts

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/kustomize"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const tektonNamespace = "tekton-pipelines"

func NewTekton(ctx *pulumi.Context, args CreateChartArgs, parent *helm.Release) (*kustomize.Directory, error) {
	/*tekton, err := yaml.NewConfigFile(ctx, "tekton",
		&yaml.ConfigFileArgs{
			File: "https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml",
			Transformations: []yaml.Transformation{
				func(state map[string]interface{}, opts ...pulumi.ResourceOption) {
					fmt.Println("Transforming Tekton")
					if state["kind"] == "Deployment" {
						spec := state["spec"].(map[string]interface{})
						spec["nodeSelector"] = GetPlacement(args.Cloud)["nodeSelector"]
						spec["tolerations"] = GetPlacement(args.Cloud)["tolerations"]
					}
				},
			},
		}, pulumi.Provider(args.Provider),
	)*/

	tekton, err := kustomize.NewDirectory(ctx, "tekton",
		kustomize.DirectoryArgs{
			Directory: pulumi.String("https://github.com/dirien/digitalocean-kubernetes-challenge/tree/main/infrastructure/manifests/tekton"),
			Transformations: []yaml.Transformation{
				func(state map[string]interface{}, opts ...pulumi.ResourceOption) {
					if state["kind"] == "Deployment" {
						spec := state["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})
						spec["nodeSelector"] = GetPlacement(args.Cloud)["nodeSelector"]
						spec["tolerations"] = GetPlacement(args.Cloud)["tolerations"]
					}
				},
			},
		}, pulumi.Provider(args.Provider))
	if err != nil {
		return nil, err
	}
	_, err = apiextensions.NewCustomResource(ctx, "tekton-dashboard-certificate", &apiextensions.CustomResourceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("tekton-dashboard-certificate"),
			Namespace: pulumi.String(tektonNamespace),
		},
		ApiVersion: pulumi.String("cert-manager.io/v1"),
		Kind:       pulumi.String("Certificate"),
		OtherFields: kubernetes.UntypedArgs{
			"spec": &pulumi.Map{
				"commonName": pulumi.String("tekton.ediri.online"),
				"dnsNames": pulumi.StringArray{
					pulumi.String("tekton.ediri.online"),
				},
				"issuerRef": &pulumi.Map{
					"name": pulumi.String("letsencrypt-staging"),
					"kind": pulumi.String("ClusterIssuer"),
				},
				"secretName": pulumi.String("tekton-dashboard-tls"),
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(parent), pulumi.Parent(tekton))
	if err != nil {
		return nil, err
	}

	_, err = v1.NewIngress(ctx, "tekton-dashboard-ingress", &v1.IngressArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name:      pulumi.String("tekton-dashboard"),
			Namespace: pulumi.String(tektonNamespace),
			Annotations: pulumi.StringMap{
				"external-dns.alpha.kubernetes.io/hostname": pulumi.String("tekton.ediri.online"),
				"external-dns.alpha.kubernetes.io/ttl":      pulumi.String("60"),
				"nginx.ingress.kubernetes.io/auth-signin":   pulumi.String("https://auth.ediri.online/oauth2/sign_in?rd=https://$host$request_uri"),
				"nginx.ingress.kubernetes.io/auth-url":      pulumi.String("http://oauth2-proxy.oauth2-proxy.svc.cluster.local/oauth2/auth"),
			},
		},
		Spec: v1.IngressSpecArgs{
			IngressClassName: pulumi.String("nginx"),
			Rules: v1.IngressRuleArray{
				v1.IngressRuleArgs{
					Host: pulumi.String("tekton.ediri.online"),
					Http: v1.HTTPIngressRuleValueArgs{
						Paths: v1.HTTPIngressPathArray{
							v1.HTTPIngressPathArgs{
								PathType: pulumi.String("ImplementationSpecific"),
								Backend: v1.IngressBackendArgs{
									Service: v1.IngressServiceBackendArgs{
										Name: pulumi.String("tekton-dashboard"),
										Port: v1.ServiceBackendPortArgs{
											Name: pulumi.String("http"),
										},
									},
								},
								Path: pulumi.String("/"),
							},
						},
					},
				},
			},
			Tls: v1.IngressTLSArray{
				v1.IngressTLSArgs{
					Hosts: pulumi.StringArray{
						pulumi.String("tekton.ediri.online"),
					},
					SecretName: pulumi.String("tekton-dashboard-tls"),
				},
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(tekton))
	if err != nil {
		return nil, err
	}

	_, err = apiextensions.NewCustomResource(ctx, "tekton-pipelines-controller-monitor", &apiextensions.CustomResourceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("tekton-pipelines-controller-monitor"),
			Namespace: pulumi.String(tektonNamespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("tekton-pipelines-controller-monitor"),
			},
		},
		ApiVersion: pulumi.String("monitoring.coreos.com/v1"),
		Kind:       pulumi.String("ServiceMonitor"),
		OtherFields: kubernetes.UntypedArgs{
			"spec": &pulumi.Map{
				"selector": &pulumi.Map{
					"matchLabels": &pulumi.Map{
						"app":                    pulumi.String("tekton-pipelines-controller"),
						"app.kubernetes.io/name": pulumi.String("controller"),
					},
				},
				"endpoints": &pulumi.Array{
					&pulumi.Map{
						"interval": pulumi.String("10s"),
						"port":     pulumi.String("http-metrics"),
					},
				},
				"namespaceSelector": &pulumi.Map{
					"matchNames": &pulumi.Array{
						pulumi.String(tektonNamespace),
					},
				},
				"jobLabel": pulumi.String("app"),
			},
		}}, pulumi.Provider(args.Provider), pulumi.Parent(tekton))
	if err != nil {
		return nil, err
	}
	_, err = apiextensions.NewCustomResource(ctx, "tekton-pipelines-webhook-monitor", &apiextensions.CustomResourceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("tekton-pipelines-webhook-monitor"),
			Namespace: pulumi.String(tektonNamespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("tekton-pipelines-webhook-monitor"),
			},
		},
		ApiVersion: pulumi.String("monitoring.coreos.com/v1"),
		Kind:       pulumi.String("ServiceMonitor"),
		OtherFields: kubernetes.UntypedArgs{
			"spec": &pulumi.Map{
				"selector": &pulumi.Map{
					"matchLabels": &pulumi.Map{
						"app":                    pulumi.String("tekton-pipelines-webhook"),
						"app.kubernetes.io/name": pulumi.String("webhook"),
					},
				},
				"endpoints": &pulumi.Array{
					&pulumi.Map{
						"interval": pulumi.String("10s"),
						"port":     pulumi.String("http-metrics"),
					},
				},
				"namespaceSelector": &pulumi.Map{
					"matchNames": &pulumi.Array{
						pulumi.String(tektonNamespace),
					},
				},
				"jobLabel": pulumi.String("app"),
			},
		}}, pulumi.Provider(args.Provider), pulumi.Parent(tekton))
	if err != nil {
		return nil, err
	}
	_, err = apiextensions.NewCustomResource(ctx, "tekton-triggers-controller-monitor", &apiextensions.CustomResourceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("tekton-triggers-controller-monitor"),
			Namespace: pulumi.String(tektonNamespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String("tekton-triggers-controller-monitor"),
			},
		},
		ApiVersion: pulumi.String("monitoring.coreos.com/v1"),
		Kind:       pulumi.String("ServiceMonitor"),
		OtherFields: kubernetes.UntypedArgs{
			"spec": &pulumi.Map{
				"selector": &pulumi.Map{
					"matchLabels": &pulumi.Map{
						"app":                    pulumi.String("tekton-triggers-controller"),
						"app.kubernetes.io/name": pulumi.String("controller"),
					},
				},
				"endpoints": &pulumi.Array{
					&pulumi.Map{
						"interval": pulumi.String("10s"),
						"port":     pulumi.String("http-metrics"),
					},
				},
				"namespaceSelector": &pulumi.Map{
					"matchNames": &pulumi.Array{
						pulumi.String(tektonNamespace),
					},
				},
				"jobLabel": pulumi.String("app"),
			},
		}}, pulumi.Provider(args.Provider), pulumi.Parent(tekton))
	if err != nil {
		return nil, err
	}
	return tekton, nil
}
