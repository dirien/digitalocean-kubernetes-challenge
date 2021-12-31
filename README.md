# DigitalOcean Kubernetes Challenge 2021

I chose to participate in the DigitalOcean Kubernetes Challenge in order to learn more about Kubernetes and to get a
better understanding of the challenges that are involved in deploying Kubernetes clusters.

## The challenge

I picked following challenge:

> Deploy a GitOps CI/CD implementation GitOps is today the way you automate deployment pipelines within Kubernetes itself, and ArgoCD  is currently one of the leading implementations. Install it to create a CI/CD solution, using tekton and kaniko for actual image building.

## The solution

### Auth0

In the folder `infrastructure/auth` I created a Pulumi Program, to deploy the Auth0 infrastructure.

Grafana and auth0
https://blog.dahanne.net/2020/04/15/integrating-auth0-oidc-oauth-2-with-authorization-groups-and-roles/

### Sealed Secrets

```
kubeseal  --controller-namespace sealed-secrets --controller-name sealed-secrets --scope cluster-wide -o yaml <github-webhook-secret.yaml >github-webhook-secret-ss.yaml
```