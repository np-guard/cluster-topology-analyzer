# GitSecure Network Topology Analyzer

### Build the project

Make sure  you have golang 1.13+ on your platform

```
$ git clone git@github.ibm.com:gitsecure/gitsecure-net-top.git
$ cd gitsecure-net-top
$ go mod download
$ make
```

### Using Docker Image

If you have access to `us.icr.io/gitsecure` registry namespace, then you can download the image and run it from there

```
$ docker run us.icr.io/gitsecure/gitsecure-nettop:1.0.0 -h
```

### Usage
```
$ ./bin/net-top -h
Usage of ./bin/net-top:
  -commitid string
    	gitsecure run id
  -dirpath string
    	input directory path
  -gitbranch string
    	git repository branch
  -giturl string
    	git repository url
```

### Example

1. Clone a sample source code repository that you want to scan
```
$ cd $HOME
$ git@github.com:nadgowdas/microservices-demo.git
```

2. Point topology analyzer to this sample repo
```
$ ./bin/net-top -dirpath $HOME/microservices-demo -commitid 9133fdc043b20be15f958339e96564eac04bed6e -gitbranch https://github.com/nadgowdas/microservices-demo -giturl matser
```

3. You can expect the result connection in following schema
```
[
    {
        "source": {
            "git_url": "",
            "git_branch": "",
            "commitid": "",
            "Resource": {
                "name": "",
                "selectors": null,
                "filepath": "",
                "kind": "",
                "image": {
                    "id": ""
                },
                "network": null,
                "Envs": null
            }
        },
        "target": {
            "git_url": "",
            "git_branch": "",
            "commitid": "",
            "Resource": {
                "name": "",
                "selectors": null,
                "filepath": "",
                "kind": "",
                "image": {
                    "id": ""
                },
                "network": null,
                "Envs": null
            }
        },
        "link": {
            "git_url": "",
            "git_branch": "",
            "commitid": "",
            "resource": {
                "name": "",
                "selectors": null,
                "filepath": "",
                "kind": "",
                "network": null
            }
        }
    }
]
```

4. Sample result
```
[
    ...
        {
        "source": {
            "git_url": "https://github.com/nadgowdas/microservices-demo",
            "git_branch": "master",
            "commitid": "9133fdc043b20be15f958339e96564eac04bed6e",
            "Resource": {
                "name": "frontend",
                "selectors": [
                    "app:frontend"
                ],
                "filepath": "",
                "kind": "Deployment",
                "image": {
                    "id": "gcr.io/google-samples/microservices-demo/frontend:v0.2.0"
                },
                "network": [
                    {
                        "container_url": 8080,
                        "protocol": ""
                    }
                ],
                "Envs": [
                    "productcatalogservice:3550",
                    "currencyservice:7000",
                    "cartservice:7070",
                    "recommendationservice:8080",
                    "shippingservice:50051",
                    "checkoutservice:5050",
                    "adservice:9555"
                ]
            }
        },
        "target": {
            "git_url": "https://github.com/nadgowdas/microservices-demo",
            "git_branch": "master",
            "commitid": "9133fdc043b20be15f958339e96564eac04bed6e",
            "Resource": {
                "name": "adservice",
                "selectors": [
                    "app:adservice"
                ],
                "filepath": "",
                "kind": "Deployment",
                "image": {
                    "id": "gcr.io/google-samples/microservices-demo/adservice:v0.2.0"
                },
                "network": [
                    {
                        "container_url": 9555,
                        "protocol": ""
                    }
                ],
                "Envs": null
            }
        },
        "link": {
            "git_url": "https://github.com/nadgowdas/microservices-demo",
            "git_branch": "master",
            "commitid": "9133fdc043b20be15f958339e96564eac04bed6e",
            "resource": {
                "name": "adservice",
                "selectors": [
                    "app:adservice"
                ],
                "filepath": "release/kubernetes-manifests.yaml",
                "kind": "Service",
                "network": [
                    {
                        "port": 9555,
                        "target_port": 9555
                    }
                ]
            }
        }
    }
]
```

### TODOs
1. Support following network/service configurations:

    a. Routes

    b. ConfigMaps

    c. Network Policies
    