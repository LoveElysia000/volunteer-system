# ==============================================================================
# 基础环境变量
# ==============================================================================
GOHOSTOS := $(shell go env GOHOSTOS)
GOPATH := $(shell go env GOPATH)
VERSION := $(shell git describe --tags --always 2>nul || echo unknown)

# 镜像与项目配置
REGISTRY ?= volunteer-system
IMG ?= volunteer-system
IMG_TAG ?= $(shell git --no-pager log -1 --format="%ad" --date=format:"%Y%m%d")-$(shell git describe --tags --always)

# ==============================================================================
# 操作系统适配 (Windows/Linux)
# ==============================================================================
ifeq ($(GOHOSTOS), windows)
    # Windows环境配置
    
    # 1. 指定 Git Bash 路径 (解决 "Program Files" 空格问题)
    # 如果你的安装路径不同，请修改这里 (注意：不要加引号，引号在后面使用时加)
    Git_Bash = C:\Program Files\Git\bin\bash.exe
    
    # 2. Protoc 编译器路径
    PROTOC = D:\Tools\protoc\bin\protoc.exe
    
    # 3. 插件绝对路径 (防止误启动系统)
    GOPATH_BIN := $(subst \,/,$(GOPATH))/bin
    PLUGIN_GO := $(GOPATH_BIN)/protoc-gen-go.exe
    PLUGIN_OPENAPI := $(GOPATH_BIN)/protoc-gen-openapi.exe

    # 4. API 文件列表
    API_PROTO_FILES = internal/api/activities.proto internal/api/login.proto internal/api/membership.proto internal/api/organization.proto internal/api/register.proto internal/api/volunteer.proto

    # 5. Protoc 执行命令封装
    PROTOC_RUN = cmd /c ""$(PROTOC)" \
        --plugin=protoc-gen-go="$(PLUGIN_GO)" \
        --plugin=protoc-gen-openapi="$(PLUGIN_OPENAPI)" \
        --proto_path=. \
        --proto_path=proto \
        --go_out=paths=source_relative:. \
        --openapi_out=fq_schema_naming=true,default_response=false:docs"

    # 6. 辅助工具命令
    # 注意："$(Git_Bash)" 引号是为了处理 Program Files 的空格
    SED_CMD = "$(Git_Bash)" -c "sed -i 's/,omitempty//g' internal/api/*.pb.go"
    INJECT_CMD = protoc-go-inject-tag -input="internal/api/*.pb.go"

else
    # Linux/Mac 环境配置
    PROTOC = protoc
    API_PROTO_FILES=$(shell find internal/api -name *.proto)
    
    PROTOC_RUN = $(PROTOC) \
        --proto_path=. \
        --proto_path=proto \
        --go_out=paths=source_relative:. \
        --openapi_out=fq_schema_naming=true,default_response=false:docs

    SED_CMD = sed -i "" -e "s/,omitempty//g" internal/api/*.pb.go
    INJECT_CMD = protoc-go-inject-tag -input="./internal/api/*.pb.go"
endif

# ==============================================================================
# Targets
# ==============================================================================

.PHONY: all install api api-single api-single-mac build run clean test docker-build

all: build

install:
	go install gorm.io/gen/tools/gentool@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/favadi/protoc-go-inject-tag@latest

# 生成 API 代码 (批量)
api:
	$(PROTOC_RUN) $(API_PROTO_FILES)
	$(SED_CMD)
	$(INJECT_CMD)

# 生成单个 Proto 文件
# 使用方法: make api-single file=internal/api/volunteer.proto
api-single:
	@if "$(file)"=="" (echo Usage: make api-single file=path/to/file.proto && exit 1)
	$(PROTOC_RUN) $(file)
	$(SED_CMD)
	$(INJECT_CMD)

# 生成单个 Proto 文件 (仅 Mac)
# 使用方法: make api-single-mac file=internal/api/volunteer.proto
api-single-mac:
	@if [ -z "$(file)" ]; then echo "Usage: make api-single-mac file=path/to/file.proto"; exit 1; fi
	$(PROTOC_RUN) $(file)
	@pb_file="$(patsubst %.proto,%.pb.go,$(file))"; \
	sed -i '' -e "s/,omitempty//g" "$$pb_file"; \
	protoc-go-inject-tag -input="$$pb_file"

build:
	go build -o volunteer-system.exe cmd/main.go

build-mac:
	go build -o volunteer-system cmd/main.go

run: build
	volunteer-system.exe -c server

clean:
	go clean
	-del /Q volunteer-system.exe

test:
	go test -v ./...

fmt:
	go fmt ./...

mod:
	go mod tidy
    
# ==============================================================================
# Database Model Generation
# ==============================================================================

# 生成数据库模型代码
models:
	@if [ ! -f "./gen.yaml" ]; then \
		echo "Error: ./gen.yaml file not found!"; \
		exit 1; \
	fi
	gentool -c "./gen.yaml"
	rm -f internal/dao/gen.go

# ==============================================================================
# Docker Actions
# ==============================================================================

docker-build:
	docker build -t $(REGISTRY)/$(IMG):$(IMG_TAG) .

docker-tag:
	docker tag $(REGISTRY)/$(IMG):$(IMG_TAG) harbor.example.com/$(REGISTRY)/$(IMG):$(IMG_TAG)

docker-push: docker-tag
	docker push harbor.example.com/$(REGISTRY)/$(IMG):$(IMG_TAG)
