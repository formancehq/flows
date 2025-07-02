VERSION 0.8

ARG core=github.com/formancehq/earthly:main
IMPORT $core AS core

FROM core+base-image

sources:
    WORKDIR src
    WORKDIR /src
    COPY go.* .
    COPY --dir pkg cmd internal .
    COPY main.go .
    SAVE ARTIFACT /src

compile:
    FROM core+builder-image
    COPY (+sources/*) /src
    WORKDIR /src
    ARG VERSION=latest
    DO --pass-args core+GO_COMPILE --VERSION=$VERSION

build-image:
    FROM core+final-image
    ENTRYPOINT ["/bin/orchestration"]
    CMD ["serve"]
    COPY (+compile/main) /bin/orchestration
    ARG REPOSITORY=ghcr.io
    ARG tag=latest
    DO core+SAVE_IMAGE --COMPONENT=orchestration --REPOSITORY=${REPOSITORY} --TAG=$tag

deploy:
    COPY (+sources/*) /src
    LET tag=$(tar cf - /src | sha1sum | awk '{print $1}')
    WAIT
        BUILD --pass-args +build-image --tag=$tag
    END
    FROM --pass-args core+vcluster-deployer-image
    RUN kubectl patch Versions.formance.com default -p "{\"spec\":{\"orchestration\": \"${tag}\"}}" --type=merge

deploy-staging:
    BUILD --pass-args core+deploy-staging

openapi:
    FROM core+base-image
    WORKDIR /src
    COPY openapi.yaml openapi.yaml
    SAVE ARTIFACT ./openapi.yaml

release:
    FROM core+builder-image
    ARG mode=local
    COPY --dir . /src
    DO core+GORELEASER --mode=$mode