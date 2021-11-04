#!/bin/bash
QPERF_BIN="../qperf-go"

function build_qperf() {
    pushd ..
    set -e
    go build qperf-go
    set +e
    popd
}