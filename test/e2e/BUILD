load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_test(
    name = "go_default_test",
    srcs = ["setup_dind_test.go"],
    args = [
        "-kindBinary=$(location @io_k8s_sigs_kind//:kind)",
        "-kubectlBinary=$(location @io_k8s_kubernetes//cmd/kubectl:kubectl)",
        "-awsProviderYAML=$(location //config:aws-provider-yaml)",
        "-clusterAPIYAML=$(location //config:cluster-api-yaml)",
    ],
    data = [
        "//config:aws-provider-yaml",
        "//config:cluster-api-yaml",
        "@io_k8s_kubernetes//cmd/kubectl:kubectl",
        "@io_k8s_sigs_kind//:kind",
    ],
    embed = [":go_default_library"],
    rundir = ".",
)

go_library(
    name = "go_default_library",
    srcs = ["e2e.go"],
    importpath = "sigs.k8s.io/cluster-api-provider-aws/test/e2e",
    visibility = ["//visibility:public"],
)
