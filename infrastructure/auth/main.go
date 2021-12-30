package main

import (
	"fmt"
	"github.com/pulumi/pulumi-auth0/sdk/v2/go/auth0"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		argoCD, err := auth0.NewClient(ctx, "argocd", &auth0.ClientArgs{
			AllowedLogoutUrls: pulumi.StringArray{
				pulumi.String("https://argocd.ediri.online"),
			},
			AllowedOrigins: pulumi.StringArray{
				pulumi.String("https://argocd.ediri.online"),
			},
			AppType: pulumi.String("regular_web"),
			Callbacks: pulumi.StringArray{
				pulumi.String("https://argocd.ediri.online/auth/callback"),
			},
			JwtConfiguration: &auth0.ClientJwtConfigurationArgs{
				Alg: pulumi.String("RS256"),
			},
		})
		if err != nil {
			return err
		}
		_, err = auth0.NewRule(ctx, "argocd-rule", &auth0.RuleArgs{
			Name:    pulumi.String("ArgoCD Group Claim"),
			Enabled: pulumi.Bool(true),
			Script:  pulumi.String(fmt.Sprintf("%v%v%v%v%v%v", "function (user, context, callback) {\n", "  var namespace = 'https://example.com/claims/';\n", "  context.idToken[namespace + \"groups\"] = user.groups;\n", "  callback(null, user, context);\n", "}\n", "\n")),
		})
		if err != nil {
			return err
		}

		ctx.Export("argo.clientId", argoCD.ClientId)
		ctx.Export("argo.clientSecret", argoCD.ClientSecret)

		grafana, err := auth0.NewClient(ctx, "grafana", &auth0.ClientArgs{
			AllowedLogoutUrls: pulumi.StringArray{
				pulumi.String("https://grafana.ediri.online"),
			},
			AllowedOrigins: pulumi.StringArray{
				pulumi.String("https://grafana.ediri.online"),
			},
			AppType: pulumi.String("regular_web"),
			Callbacks: pulumi.StringArray{
				pulumi.String("https://grafana.ediri.online/auth/callback"),
				pulumi.String("https://grafana.ediri.online/login/generic_oauth"),
			},
			JwtConfiguration: &auth0.ClientJwtConfigurationArgs{
				Alg: pulumi.String("RS256"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("grafana.clientId", grafana.ClientId)
		ctx.Export("grafana.clientSecret", grafana.ClientSecret)

		oauth2, err := auth0.NewClient(ctx, "oauth2", &auth0.ClientArgs{
			AllowedLogoutUrls: pulumi.StringArray{
				pulumi.String("https://auth.ediri.online"),
			},
			AllowedOrigins: pulumi.StringArray{
				pulumi.String("https://auth.ediri.online"),
			},
			AppType: pulumi.String("regular_web"),
			Callbacks: pulumi.StringArray{
				pulumi.String("https://auth.ediri.online/oauth2/callback"),
			},
			JwtConfiguration: &auth0.ClientJwtConfigurationArgs{
				Alg: pulumi.String("RS256"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("oauth2.clientId", oauth2.ClientId)
		ctx.Export("oauth2.clientSecret", oauth2.ClientSecret)

		return nil
	})
}
