package charts

import (
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func NewExternalDNS(ctx *pulumi.Context, args CreateChartArgs) (*helm.Release, error) {
	externalDNSNS, err := v1.NewNamespace(ctx, "external-dns", &v1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("external-dns"),
		},
	}, pulumi.Provider(args.Provider))
	if err != nil {
		return nil, err
	}

	doToken, err := v1.NewSecret(ctx, "external-dns-credentials", &v1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("external-dns-credentials"),
			Namespace: externalDNSNS.Metadata.Name(),
		},
		StringData: pulumi.StringMap{
			"do_token": pulumi.String(config.Get(ctx, "do_token")),
		},
		Type: pulumi.String("Opaque"),
	}, pulumi.Provider(args.Provider), pulumi.Parent(externalDNSNS))
	if err != nil {
		return nil, err
	}

	externalDNS, err := helm.NewRelease(ctx, "external-dns", &helm.ReleaseArgs{
		Name:            pulumi.String("external-dns"),
		Chart:           pulumi.String("external-dns"),
		Version:         pulumi.String("1.7.0"),
		Namespace:       externalDNSNS.Metadata.Name(),
		CreateNamespace: pulumi.Bool(false),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://kubernetes-sigs.github.io/external-dns"),
		},
		Values: pulumi.Map{
			"env": pulumi.Array{
				pulumi.Map{
					"name": pulumi.String("DO_TOKEN"),
					"valueFrom": pulumi.Map{
						"secretKeyRef": pulumi.Map{
							"name": doToken.Metadata.Name(),
							"key":  pulumi.String("do_token"),
						},
					},
				},
			},
			"serviceMonitor": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"additionalLabels": pulumi.Map{
					"app": pulumi.String("external-dns"),
				},
			},
			"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
			"tolerations":  GetPlacement(args.Cloud)["tolerations"],
			"provider":     pulumi.String("digitalocean"),
			"domainFilters": pulumi.Array{
				pulumi.String("ediri.online"),
			},
			"sources": pulumi.Array{
				pulumi.String("ingress"),
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(externalDNSNS))
	if err != nil {
		return nil, err
	}
	return externalDNS, nil
}
