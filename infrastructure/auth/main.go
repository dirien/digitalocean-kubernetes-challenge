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

		ctx.Export("clientId", argoCD.ClientId)
		ctx.Export("clientSecret", argoCD.ClientSecret)
		return nil
	})
}
