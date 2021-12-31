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

func NewOAuth2(ctx *pulumi.Context, args CreateChartArgs, parent *helm.Release) (*helm.Release, error) {
	oauth2ProxyNS, err := v1.NewNamespace(ctx, "oauth2-proxy-ns", &v1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("oauth2-proxy"),
		},
	}, pulumi.Provider(args.Provider))
	if err != nil {
		return nil, err
	}

	_, err = apiextensions.NewCustomResource(ctx, "oauth2-certificate", &apiextensions.CustomResourceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("oauth2-certificate"),
			Namespace: oauth2ProxyNS.Metadata.Name(),
		},
		ApiVersion: pulumi.String("cert-manager.io/v1"),
		Kind:       pulumi.String("Certificate"),
		OtherFields: kubernetes.UntypedArgs{
			"spec": &pulumi.Map{
				"commonName": pulumi.String("auth.ediri.online"),
				"dnsNames": pulumi.StringArray{
					pulumi.String("auth.ediri.online"),
				},
				"issuerRef": &pulumi.Map{
					"name": pulumi.String("letsencrypt-staging"),
					"kind": pulumi.String("ClusterIssuer"),
				},
				"secretName": pulumi.String("oauth-tls"),
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(parent), pulumi.Parent(oauth2ProxyNS))
	if err != nil {
		return nil, err
	}

	auth0Domain := config.Get(ctx, "auth0Domain")

	oauth, err := helm.NewRelease(ctx, "oauth2-proxy", &helm.ReleaseArgs{
		Name:      pulumi.String("oauth2-proxy"),
		Chart:     pulumi.String("oauth2-proxy"),
		Version:   pulumi.String("5.0.6"),
		Namespace: oauth2ProxyNS.Metadata.Name(),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://oauth2-proxy.github.io/manifests"),
		},
		Values: pulumi.Map{
			"metrics": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"serviceMonitor": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
			},
			"config": pulumi.Map{
				"clientID":     args.Auth.GetStringOutput(pulumi.String("oauth2.clientId")),
				"clientSecret": args.Auth.GetStringOutput(pulumi.String("oauth2.clientSecret")),
			},
			"extraArgs": pulumi.Map{
				"provider":              pulumi.String("oidc"),
				"provider-display-name": pulumi.String("auth0"),
				"redirect-url":          pulumi.String("https://auth.ediri.online/oauth2/callback"),
				"oidc-issuer-url":       pulumi.String(fmt.Sprintf("https://%s/", auth0Domain)),
				"cookie-expire":         pulumi.String("24h0m0s"),
				"whitelist-domain":      pulumi.String(".ediri.online"),
				"email-domain":          pulumi.String("*"),
				"cookie-refresh":        pulumi.String("0h60m0s"),
				"cookie-domain":         pulumi.String(".ediri.online"),
			},
			"nodeSelector": GetPlacement(args.Cloud)["nodeSelector"],
			"tolerations":  GetPlacement(args.Cloud)["tolerations"],
			"ingress": pulumi.Map{
				"enabled":   pulumi.Bool(true),
				"className": pulumi.String("nginx"),
				"annotations": pulumi.Map{
					"external-dns.alpha.kubernetes.io/hostname": pulumi.String("auth.ediri.online"),
					"external-dns.alpha.kubernetes.io/ttl":      pulumi.String("60"),
				},
				"hosts": pulumi.StringArray{
					pulumi.String("auth.ediri.online"),
				},
				"tls": pulumi.Array{
					pulumi.Map{
						"hosts": pulumi.StringArray{
							pulumi.String("auth.ediri.online"),
						},
						"secretName": pulumi.String("oauth-tls"),
					},
				},
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(oauth2ProxyNS))
	if err != nil {
		return nil, err
	}
	return oauth, nil
}
