GO=go

export CGO_CPPFLAGS += -I$(shell pwd)/third_party/keystone/include -I$(shell pwd)/third_party/capstone/include
export CGO_LDFLAGS += -lkeystone -lstdc++ -lm -L$(shell pwd)/third_party/keystone/build/llvm/lib -L$(shell pwd)/third_party/capstone/

KEYSTONE_LIB=./third_party/keystone/build/llvm/lib/libkeystone.a
CAPSTONE_LIB=./third_party/capstone/libcapstone.a

.PHONY: all binch static

all: binch

$(KEYSTONE_LIB):
	./build-keystone.sh

$(CAPSTONE_LIB):
	./build-capstone.sh

binch: $(KEYSTONE_LIB) $(CAPSTONE_LIB)
	$(GO) mod vendor
	$(GO) build -mod vendor -o bin/binch ./cmd/binch

static: $(KEYSTONE_LIB) $(CAPSTONE_LIB)
	$(GO) mod vendor
	$(GO) build -mod vendor -a -tags netgo -ldflags '-w -extldflags "-static"' -o bin/binch ./cmd/binch
	strip ./bin/binch
