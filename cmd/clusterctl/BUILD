load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "sigs.k8s.io/cluster-api-provider-aws/cmd/clusterctl",
    visibility = ["//visibility:private"],
    deps = [
        "//pkg/cloud/aws/actuators/cluster:go_default_library",
        "//vendor/sigs.k8s.io/cluster-api/cmd/clusterctl/cmd:go_default_library",
        "//vendor/sigs.k8s.io/cluster-api/pkg/apis/cluster/common:go_default_library",
    ],
)

go_binary(
    name = "clusterctl",
    embed = [":go_default_library"],
    pure = "on",
    visibility = ["//visibility:public"],
)
