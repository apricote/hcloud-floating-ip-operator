
CODE_GENERATOR_IMAGE := slok/kube-code-generator:v1.9.1
DIRECTORY := $(PWD)
CODE_GENERATOR_PACKAGE := github.com/apricote/hcloud-floating-ip-operator/

generate:
	docker run --rm -it \
	-v $(DIRECTORY):/go/src/$(CODE_GENERATOR_PACKAGE) \
	-e PROJECT_PACKAGE=$(CODE_GENERATOR_PACKAGE) \
	-e CLIENT_GENERATOR_OUT=$(CODE_GENERATOR_PACKAGE)/pkg/client/k8s \
	-e APIS_ROOT=$(CODE_GENERATOR_PACKAGE)/pkg/apis \
	-e GROUPS_VERSION="hcloud:v1alpha1" \
	-e GENERATION_TARGETS="deepcopy,client" \
	$(CODE_GENERATOR_IMAGE)