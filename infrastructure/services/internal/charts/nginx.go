package charts

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewIngressNginx(ctx *pulumi.Context, args CreateChartArgs) (*helm.Release, error) {

	ingressNginx, err := helm.NewRelease(ctx, "ingress-nginx", &helm.ReleaseArgs{
		Name:            pulumi.String("ingress-nginx"),
		Chart:           pulumi.String("ingress-nginx"),
		Version:         pulumi.String("4.0.13"),
		Namespace:       pulumi.String("ingress-nginx"),
		CreateNamespace: pulumi.Bool(true),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://kubernetes.github.io/ingress-nginx"),
		},
		Values: pulumi.Map{
			"serviceAccount": pulumi.Map{
				"automountServiceAccountToken": pulumi.Bool(true),
			},
			"controller": pulumi.Map{
				"metrics": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"serviceMonitor": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
				// CVE-2021-25742-nginx-ingress-snippet-annotation-vulnerability
				// https://www.accurics.com/blog/security-blog/kubernetes-security-preventing-secrets-exfiltration-cve-2021-25742/
				"allowSnippetAnnotations": pulumi.Bool(false),
				"nodeSelector":            GetPlacement(args.Cloud)["nodeSelector"],
				"tolerations":             GetPlacement(args.Cloud)["tolerations"],
			},
		},
	}, pulumi.Provider(args.Provider))
	if err != nil {
		return nil, err
	}
	return ingressNginx, nil

}
