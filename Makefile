# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

generate: generate-controller generate-groups rbacs manifests fmt

#generate helm documentation
docs: helm-docs
	$(HELM_DOCS) -t deployments/liqo/README.gotmpl deployments/liqo
	cat docs/templates/helm_reference_header.md deployments/liqo/README.md > docs/pages/installation/chart_values.md

#run all tests
test: unit e2e

# Check if test image exists
test-container:
ifeq (, $(shell docker image ls | grep liqo-test))
	@{ \
	docker build -t liqo-test -f build/liqo-test/Dockerfile . ; \
	}
endif

# Run unit tests
unit: test-container
	docker run --privileged=true --mount type=bind,src=$(shell pwd),dst=/go/src/liqo -w /go/src/liqo --rm liqo-test

# Install LIQO into a cluster
install: manifests
	./install.sh

# Uninstall LIQO from a cluster
uninstall: manifests
	./install.sh --uninstall

# Uninstall LIQO from a cluster with purge flag
purge: manifests
	./install.sh --uninstall --purge

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	rm -f deployments/liqo/crds/*
	$(CONTROLLER_GEN) crd paths="./apis/..." output:crd:artifacts:config=deployments/liqo/crds

#Generate RBAC for each controller
rbacs: controller-gen
	rm -f deployments/liqo/files/*
	$(CONTROLLER_GEN) paths="./internal/liqonet/route-operator" rbac:roleName=liqo-route output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-route-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-route-ClusterRole.yaml deployments/liqo/files/liqo-route-Role.yaml
	$(CONTROLLER_GEN) paths="./internal/liqonet/tunnel-operator" rbac:roleName=liqo-gateway output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-gateway-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-gateway-ClusterRole.yaml deployments/liqo/files/liqo-gateway-Role.yaml
	$(CONTROLLER_GEN) paths="./internal/liqonet/network-manager/..." rbac:roleName=liqo-network-manager output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-network-manager-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-network-manager-ClusterRole.yaml deployments/liqo/files/liqo-network-manager-Role.yaml
	$(CONTROLLER_GEN) paths="./internal/crdReplicator" rbac:roleName=liqo-crd-replicator output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-crd-replicator-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-crd-replicator-ClusterRole.yaml deployments/liqo/files/liqo-crd-replicator-Role.yaml
	$(CONTROLLER_GEN) paths="./pkg/discoverymanager" rbac:roleName=liqo-discovery output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-discovery-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-discovery-ClusterRole.yaml deployments/liqo/files/liqo-discovery-Role.yaml
	$(CONTROLLER_GEN) paths="./internal/auth-service" rbac:roleName=liqo-auth-service output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-auth-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-auth-ClusterRole.yaml deployments/liqo/files/liqo-auth-Role.yaml
	$(CONTROLLER_GEN) paths="./pkg/mutate" rbac:roleName=liqo-webhook output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-webhook-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-webhook-ClusterRole.yaml deployments/liqo/files/liqo-webhook-Role.yaml
	$(CONTROLLER_GEN) paths="./pkg/peering-roles/basic" rbac:roleName=liqo-remote-peering-basic output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-remote-peering-basic-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-remote-peering-basic-ClusterRole.yaml
	$(CONTROLLER_GEN) paths="./pkg/peering-roles/incoming" rbac:roleName=liqo-remote-peering-incoming output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-remote-peering-incoming-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-remote-peering-incoming-ClusterRole.yaml
	$(CONTROLLER_GEN) paths="./pkg/peering-roles/outgoing" rbac:roleName=liqo-remote-peering-outgoing output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-remote-peering-outgoing-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-remote-peering-outgoing-ClusterRole.yaml
	$(CONTROLLER_GEN) paths="./pkg/liqo-controller-manager/..." rbac:roleName=liqo-controller-manager output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-controller-manager-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-controller-manager-ClusterRole.yaml deployments/liqo/files/liqo-controller-manager-Role.yaml
	$(CONTROLLER_GEN) paths="./pkg/virtualKubelet/roles/local" rbac:roleName=liqo-virtual-kubelet-local output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-virtual-kubelet-local-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-virtual-kubelet-local-ClusterRole.yaml
	$(CONTROLLER_GEN) paths="./pkg/virtualKubelet/roles/remote" rbac:roleName=liqo-virtual-kubelet-remote output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-virtual-kubelet-remote-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-virtual-kubelet-remote-ClusterRole.yaml
	$(CONTROLLER_GEN) paths="./cmd/uninstaller" rbac:roleName=liqo-pre-delete output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-pre-delete-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-pre-delete-ClusterRole.yaml
	$(CONTROLLER_GEN) paths="./cmd/metric-agent" rbac:roleName=liqo-metric-agent output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/liqo/files/liqo-metric-agent-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' &&  sed -i -n '/rules/,$$p' deployments/liqo/files/liqo-metric-agent-ClusterRole.yaml

# Install gci if not available
gci:
ifeq (, $(shell which gci))
	@go install github.com/daixiang0/gci@v0.2.9
GCI=$(GOBIN)/gci
else
GCI=$(shell which gci)
endif

# Install addlicense if not available
addlicense:
ifeq (, $(shell which addlicense))
	@go install github.com/google/addlicense@v1.0.0
ADDLICENSE=$(GOBIN)/addlicense
else
ADDLICENSE=$(shell which addlicense)
endif

# Run go fmt against code
fmt: gci addlicense
	go mod tidy
	go fmt ./...
	find . -type f -name '*.go' -a ! -name '*zz_generated*' -exec $(GCI) -local github.com/liqotech/liqo -w {} \;
	find . -type f -name '*.go' -exec $(ADDLICENSE) -l apache -c "The Liqo Authors" -y "2019-$(shell date +%Y)" {} \;

# Install golangci-lint if not available
golangci-lint:
ifeq (, $(shell which golangci-lint))
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.0
GOLANGCILINT=$(GOBIN)/golangci-lint
else
GOLANGCILINT=$(shell which golangci-lint)
endif

lint: golangci-lint
	 $(GOLANGCILINT) run --new

generate-controller: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./apis/..."

generate-groups:
	if [ ! -d  "hack/code-generator" ]; then \
		git clone --depth 1 -b v0.22.3 https://github.com/kubernetes/code-generator.git hack/code-generator; \
	fi
	rm -rf pkg/client
	hack/code-generator/generate-groups.sh client,lister,informer \
		github.com/liqotech/liqo/pkg/client github.com/liqotech/liqo/apis \
		"virtualkubelet:v1alpha1" \
		--output-base ./ \
		-h hack/boilerplate.go.txt && \
	mv github.com/liqotech/liqo/pkg/client pkg/ && \
	rm -rf github.com

# Generate gRPC files
grpc: protoc
	$(PROTOC) --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/liqonet/ipam/ipam.proto
	$(PROTOC) --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/liqo-controller-manager/resource-request-controller/resource-monitors/resource-reader.proto

protoc:
ifeq (, $(shell which protoc))
	@{ \
	PB_REL="https://github.com/protocolbuffers/protobuf/releases" ;\
	version=3.15.5 ;\
	arch=x86_64 ;\
	curl -LO $${PB_REL}/download/v$${version}/protoc-$${version}-linux-$${arch}.zip ;\
	unzip protoc-$${version}-linux-$${arch}.zip -d $${HOME}/.local ;\
	rm protoc-$${version}-linux-$${arch}.zip ;\
	PROTOC_TMP_DIR=$$(mktemp -d) ;\
	cd $$PROTOC_TMP_DIR ;\
	go mod init tmp ;\
	go get google.golang.org/protobuf/cmd/protoc-gen-go ;\
	go get google.golang.org/grpc/cmd/protoc-gen-go-grpc ;\
	rm -rf $$PROTOC_TMP_DIR ;\
	}
endif
PROTOC=$(shell which protoc)

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

helm-docs:
ifeq (, $(shell which helm-docs))
	@{ \
	set -e ;\
	HELM_DOCS_TMP_DIR=$$(mktemp -d) ;\
	cd $$HELM_DOCS_TMP_DIR ;\
	version=1.5.0 ;\
    arch=x86_64 ;\
    echo  $$HELM_DOCS_PATH ;\
    echo https://github.com/norwoodj/helm-docs/releases/download/v$${version}/helm-docs_$${version}_linux_$${arch}.tar.gz ;\
    curl -LO https://github.com/norwoodj/helm-docs/releases/download/v$${version}/helm-docs_$${version}_linux_$${arch}.tar.gz ;\
    tar -zxvf helm-docs_$${version}_linux_$${arch}.tar.gz ;\
    mv helm-docs $(GOBIN)/helm-docs ;\
	rm -rf $$HELM_DOCS_TMP_DIR ;\
	}
HELM_DOCS=$(GOBIN)/helm-docs
else
HELM_DOCS=$(shell which helm-docs)
endif

# Set the steps for the e2e tests
E2E_TARGETS = e2e-dir \
	e2e-liqoctl \
	e2e-infra \
	installer/liqoctl/setup \
	installer/liqoctl/peer \
	e2e/postinstall \
	e2e/cruise \
	installer/liqoctl/unpeer \
	installer/liqoctl/uninstall \
	e2e/postuninstall

# Export these variables before to run the e2e tests

# export CLUSTER_NUMBER=2
# export K8S_VERSION=v1.21.1
# export CNI=kindnet
# export TMPDIR=$(mktemp -d)
# export BINDIR=${TMPDIR}/bin
# export TEMPLATE_DIR=${PWD}/test/e2e/pipeline/infra/kind
# export NAMESPACE=liqo
# export KUBECONFIGDIR=${TMPDIR}/kubeconfigs
# export LIQO_VERSION=3e060bc36ffb1a88b988a7e948de2b045ba2e8ce
# export INFRA=kind
# export LIQOCTL=${BINDIR}/liqoctl
# export POD_CIDR_OVERLAPPING=false
# export TEMPLATE_FILE=cluster-templates.yaml.tmpl

# Run e2e tests
e2e: $(E2E_TARGETS)

e2e-dir:
	mkdir -p "${BINDIR}"

e2e-liqoctl:
	go build -o "${BINDIR}/liqoctl" ./cmd/liqoctl

e2e-infra:
	${PWD}/test/e2e/pipeline/infra/${INFRA}/pre-requirements.sh
	${PWD}/test/e2e/pipeline/infra/${INFRA}/clean.sh
	${PWD}/test/e2e/pipeline/infra/${INFRA}/setup.sh

installer/%:
	${PWD}/test/e2e/pipeline/$@.sh

e2e/%:
	go test ${PWD}/test/$@/... -count=1 -timeout=20m
