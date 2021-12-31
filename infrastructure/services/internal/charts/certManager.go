package charts

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewCertManager(ctx *pulumi.Context, args CreateChartArgs) (*helm.Release, error) {
	certManager, err := helm.NewRelease(ctx, "jetstack", &helm.ReleaseArgs{
		Name:            pulumi.String("cert-manager"),
		Chart:           pulumi.String("cert-manager"),
		Version:         pulumi.String("v1.6.1"),
		Namespace:       pulumi.String("cert-manager"),
		CreateNamespace: pulumi.Bool(true),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://charts.jetstack.io"),
		},
		Values: pulumi.Map{
			"prometheus": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"servicemonitor": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
			},
			"serviceAccount": pulumi.Map{
				"automountServiceAccountToken": pulumi.Bool(true),
			},
			"installCRDs":  pulumi.Bool(true),
			"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
			"tolerations":  GetPlacement(args.Cloud)["tolerations"],
			"webhook": pulumi.Map{
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
			},
			"cainjector": pulumi.Map{
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
			},
			"startupapicheck": pulumi.Map{
				"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":  GetPlacement(args.Cloud)["tolerations"],
			},
		},
	}, pulumi.Provider(args.Provider))
	if err != nil {
		return nil, err
	}
	_, err = apiextensions.NewCustomResource(ctx, "letsencrypt-staging", &apiextensions.CustomResourceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("letsencrypt-staging"),
		},
		ApiVersion: pulumi.String("cert-manager.io/v1"),
		Kind:       pulumi.String("ClusterIssuer"),
		OtherFields: kubernetes.UntypedArgs{
			"spec": &pulumi.Map{
				"acme": pulumi.Map{
					"server": pulumi.String("https://acme-staging-v02.api.letsencrypt.org/directory"),
					"email":  pulumi.String("info@ediri.de"),
					"privateKeySecretRef": pulumi.StringMap{
						"name": pulumi.String("letsencrypt-staging"),
					},
					"solvers": pulumi.Array{
						pulumi.Map{
							"http01": pulumi.StringMapMap{
								"ingress": pulumi.StringMap{
									"class": pulumi.String("nginx"),
								},
							},
						},
					},
				},
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(certManager))
	if err != nil {
		return nil, err
	}
	return certManager, nil
}
