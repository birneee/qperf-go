#!/bin/bash
QPERF_BIN="../qperf-go"

function build_qperf() {
  (cd .. ; go build qperf-go)
}