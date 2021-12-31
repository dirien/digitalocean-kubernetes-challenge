package charts

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewSealedSecrets(ctx *pulumi.Context, args CreateChartArgs) (*helm.Release, error) {
	sealedSecrets, err := helm.NewRelease(ctx, "sealed-secrets", &helm.ReleaseArgs{
		Name:  pulumi.String("sealed-secrets"),
		Chart: pulumi.String("sealed-secrets"),
		//https://github.com/bitnami-labs/sealed-secrets/issues/694
		Version:         pulumi.String("1.16.1"),
		Namespace:       pulumi.String("sealed-secrets"),
		CreateNamespace: pulumi.Bool(true),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://bitnami-labs.github.io/sealed-secrets"),
		},
		Values: pulumi.Map{
			//https://github.com/argoproj/argo-cd/issues/5991#issuecomment-890541970,
			"commandArgs": pulumi.StringArray{
				pulumi.String("--update-status"),
			},
			"serviceMonitor": pulumi.Map{
				"create": pulumi.Bool(true),
			},
			"dashboards": pulumi.Map{
				"create": pulumi.Bool(true),
				"labels": pulumi.Map{
					"grafana_dashboard": pulumi.String("1"),
				},
			},
			"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
			"tolerations":  GetPlacement(args.Cloud)["tolerations"],
		},
	}, pulumi.Provider(args.Provider))
	if err != nil {
		return nil, err
	}
	return sealedSecrets, nil
}
