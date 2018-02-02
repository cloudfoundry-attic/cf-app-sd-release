# CF App Service Discovery Release (EXPERIMENTAL)


## High Level Overview

### Problem we are trying to solve
Application Developers who want to use container to container networking today are required to bring their own service discovery. While we have provided examples with Eureka and Amalgam8, we have received user feedback that usage of c2c is very difficult, with some common themes emerging:
Polyglot microservices written in languages/frameworks other than Java/Spring cannot easily use Eureka
Clustering applications have a requirement to address individual instances
Additional VMs need to be deployed and managed to provide external service discovery

In order to support all types of apps, languages and frameworks, we plan to build service discovery for c2c into the platform. With this feature, users will no longer have to bring their own service discovery. 

### App Developer Experience

When a user pushes an app, their app is automatically given an internal DNS entry, e.g.
```
app-guid.apps.internal
```
This DNS entry will automatically load balance between all instances of this app. For the time being, your tld will always be `apps.internal`.

Each instance of the application also receives a specific DNS entry, e.g.
```
instance-index.app-guid.apps.internal
```
that you can use to access one instance of the app, so it will not load balance between all instances.

### Interaction with Policy

By default, apps cannot talk to each other over cf networking. In order for an app to talk to another app, you must still set a policy allowing access. 

### Example usage

```
cf push consumer-app --no-start
cf push server-app

cf add-network-policy consumer-app --destination-app server-app --port 8080 --protocol tcp

cf set-env consumer SERVER_HOSTNAME "$(cf app --guid server-app).apps.internal"
cf start consumer-app
```

You can run `cf add-network-policy` even after both apps are started, and you don't need to restart the apps for the policy to start working.

From consumer-app, the following will work:
```
curl "$SERVER_HOSTNAME.apps.internal:8080"
```

## Architecture

### Architecture Diagram
![](architecture-diagram.png)

## How To Deploy

To add service discovery to cf-deployment, include the following experimental ops-files:
- [Service Discovery ops file](https://github.com/cloudfoundry/cf-deployment/blob/release-candidate/operations/experimental/enable-service-discovery.yml)
- [BOSH DNS ops file](https://github.com/cloudfoundry/cf-deployment/blob/release-candidate/operations/experimental/use-bosh-dns.yml)
- [BOSH DNS for containers ops file](https://github.com/cloudfoundry/cf-deployment/blob/release-candidate/operations/experimental/use-bosh-dns-for-containers.yml)

### Experimental Ops File for cf-deployment

* Pull down your current manifest with 
```
bosh manifest > /tmp/{env}-manifest.yml
```

* Update your deployment with the ops files 
``` bash
bosh deploy /tmp/{env}-manifest.yml \
  -o ~/workspace/cf-deployment/operations/experimental/use-bosh-dns-for-containers.yml \
  -o ~/workspace/cf-deployment/operations/experimental/use-bosh-dns.yml \
  -o ~/workspace/cf-deployment/operations/experimental/enable-service-discovery.yml \
  --vars-store ~/workspace/cf-networking-deployments/environments/{env}/vars-store.yml
```

See [opsfile](https://github.com/cloudfoundry/cf-deployment/blob/release-candidate/operations/experimental/enable-service-discovery.yml) and our [pipeline](ci/pipelines/cf-app-sd.yml) that uses this opsfile.

### Debugging problems

* To change logging for service-discovery-controller, ssh onto the VM holding the service-discovery-controller and make a request to the log-level server:
```bash
curl -X POST -d 'debug' localhost:8055/log-level
```
where `8055` is the default value of `service-discovery-controller.log_level_port`.

To switch back to `info` logging:
```bash
curl -X POST -d 'info' localhost:8055/log-level
```

* To change logging for bosh-dns-adapter, ssh onto the VM holding the bosh-dns-adapter and make a request to the log-level server:
```bash
curl -X POST -d 'debug' localhost:8066/log-level
```

To switch back to `info` logging:
```bash
curl -X POST -d 'info' localhost:8066/log-level
```


### Deploying to BOSH-lite

Run the [`scripts/deploy-to-bosh-lite`](scripts/deploy-to-bosh-lite) script.

To deploy you will need [cf-networking-release](https://github.com/cloudfoundry/cf-networking-release), [bosh-deployment](https://github.com/cloudfoundry/bosh-deployment), and [cf-deployment](https://github.com/cloudfoundry/cf-deployment).
