# ONNX

This function only need onnx-runtime lib, and the lib is support by [ort](https://github.com/yalue/onnxruntime_go) ,so you can use it directly rather than install C API like Tensorflow Lite.

Support onnxruntime.so is enough.

## Version requirement

The bundled shared libraries (`onnxruntime.so`, `onnxruntime_arm64.so`,
`onnxruntime_arm64.dylib`) are ONNX Runtime 1.17.1, which exposes C API version
17. The Go binding is pinned to `github.com/yalue/onnxruntime_go v1.9.0` because
that is the newest release whose header declares `ORT_API_VERSION 17`; later
releases ask the runtime for API version 20 and above and fail at
`InitializeEnvironment` with:

```text
The requested API version [31] is not available, only API versions [1, 17]
```

When upgrading the shared libraries, upgrade `onnxruntime_go` in the same
commit and keep the two in sync: the API version the Go binding is compiled
against must be less than or equal to the one the shared library reports.
