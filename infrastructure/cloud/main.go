package main

import (
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		dokRegion := config.Get(ctx, "region")
		kubernetesCluster, err := digitalocean.NewKubernetesCluster(ctx, "digitalocean-kubernetes-challenge", &digitalocean.KubernetesClusterArgs{
			Name:        pulumi.String("digitalocean-kubernetes-challenge"),
			Ha:          pulumi.Bool(false),
			Version:     pulumi.String(config.Get(ctx, "version")),
			Region:      pulumi.String(dokRegion),
			AutoUpgrade: pulumi.Bool(true),
			MaintenancePolicy: &digitalocean.KubernetesClusterMaintenancePolicyArgs{
				Day:       pulumi.String("sunday"),
				StartTime: pulumi.String("00:00"),
			},
			Tags: pulumi.StringArray{
				pulumi.String("do-kubernetes-challenge"),
				pulumi.String("ci-cd"),
				pulumi.String("argoCD"),
				pulumi.String("tekton"),
			},
			NodePool: &digitalocean.KubernetesClusterNodePoolArgs{
				Name:      pulumi.String("base-node-pool"),
				Size:      pulumi.String(config.Get(ctx, "size-base-node-pool")),
				AutoScale: pulumi.Bool(true),
				MinNodes:  pulumi.Int(1),
				MaxNodes:  pulumi.Int(3),
				Tags: pulumi.StringArray{
					pulumi.String("base"),
				},
			},
		})
		if err != nil {
			return err
		}
		toolsNodePool, err := digitalocean.NewKubernetesNodePool(ctx, "tools-node-pool", &digitalocean.KubernetesNodePoolArgs{
			ClusterId: kubernetesCluster.ID(),
			Name:      pulumi.String("tools-node-pool"),
			Size:      pulumi.String(config.Get(ctx, "size-tools-node-pool")),
			AutoScale: pulumi.Bool(true),
			MinNodes:  pulumi.Int(1),
			MaxNodes:  pulumi.Int(3),
			Tags: pulumi.StringArray{
				pulumi.String("tools"),
			},
			Taints: digitalocean.KubernetesNodePoolTaintArray{
				&digitalocean.KubernetesNodePoolTaintArgs{
					Key:    pulumi.String("tools"),
					Value:  pulumi.String("true"),
					Effect: pulumi.String("NoSchedule"),
				},
			},
		})
		if err != nil {
			return err
		}
		_, err = digitalocean.NewKubernetesNodePool(ctx, "workload-node-pool", &digitalocean.KubernetesNodePoolArgs{
			ClusterId: kubernetesCluster.ID(),
			Name:      pulumi.String("workload-node-pool"),
			Size:      pulumi.String(config.Get(ctx, "size-workload-node-pool")),
			AutoScale: pulumi.Bool(true),
			MinNodes:  pulumi.Int(1),
			MaxNodes:  pulumi.Int(3),
			Tags: pulumi.StringArray{
				pulumi.String("workload"),
			},
			Taints: digitalocean.KubernetesNodePoolTaintArray{
				&digitalocean.KubernetesNodePoolTaintArgs{
					Key:    pulumi.String("workload"),
					Value:  pulumi.String("true"),
					Effect: pulumi.String("NoSchedule"),
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("cluster", kubernetesCluster.Name)
		ctx.Export("toolsNodePoolName", toolsNodePool.Name)
		ctx.Export("kubeconfig", pulumi.ToSecret(kubernetesCluster.KubeConfigs.ToKubernetesClusterKubeConfigArrayOutput().Index(pulumi.Int(0)).RawConfig()))

		bucket, err := digitalocean.NewSpacesBucket(ctx, "loki-bucket", &digitalocean.SpacesBucketArgs{
			Name:   pulumi.String("loki-bucket"),
			Region: pulumi.String(dokRegion),
		})
		if err != nil {
			return err
		}

		ctx.Export("loki.bucket.Name", bucket.Name)
		ctx.Export("loki.bucket.BucketDomainName", bucket.BucketDomainName)
		ctx.Export("loki.bucket.region", bucket.Region)

		return nil
	})
}
