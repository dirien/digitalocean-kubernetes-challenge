package main

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"services/internal/charts"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		cloud, err := pulumi.NewStackReference(ctx, "dirien/cloud/do-kubernetes-challenge", nil)
		if err != nil {
			return err
		}

		auth, err := pulumi.NewStackReference(ctx, "dirien/auth/do-kubernetes-challenge", nil)
		if err != nil {
			return err
		}

		provider, err := kubernetes.NewProvider(ctx, "kubernetes", &kubernetes.ProviderArgs{
			Kubeconfig: cloud.GetStringOutput(pulumi.String("kubeconfig")),
		})
		if err != nil {
			return err
		}

		args := charts.CreateChartArgs{
			Cloud:    cloud,
			Auth:     auth,
			Provider: provider,
		}

		_, err = charts.NewIngressNginx(ctx, args)
		if err != nil {
			return err
		}
		_, err = charts.NewExternalDNS(ctx, args)
		if err != nil {
			return err
		}
		certManager, err := charts.NewCertManager(ctx, args)
		if err != nil {
			return err
		}
		_, err = charts.NewArgoCD(ctx, args, certManager)
		if err != nil {
			return err
		}
		_, err = charts.NewMonitoring(ctx, args, certManager)
		if err != nil {
			return err
		}
		/*_, err = charts.NewCole(ctx, args, monitoring)
		if err != nil {
			return err
		}*/
		_, err = charts.NewTekton(ctx, args, certManager)
		if err != nil {
			return err
		}
		_, err = charts.NewOAuth2(ctx, args, certManager)
		if err != nil {
			return err
		}
		_, err = charts.NewSealedSecrets(ctx, args)
		if err != nil {
			return err
		}
		return nil
	})
}
