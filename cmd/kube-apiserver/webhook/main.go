package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func main() {
	// 注册 /mutate 路由
	http.HandleFunc("/mutate", mutate)
	// 启动 webhook 服务
	panic(http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil))
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func mutate(w http.ResponseWriter, r *http.Request) {
	fmt.Println("收到来自 kube-apiserver 的请求.")

	// 解析 body 为 Pod 对象
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	deserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	ar := admissionv1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var pod corev1.Pod
	if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 执行变更操作，添加一个注释
	pod.ObjectMeta.Annotations["mutate"] = "欢迎关注公众号：gopher云原生"

	// 构造 Patch 对象
	patch := []patchOperation{
		{
			Op:    "add",
			Path:  "/metadata/annotations",
			Value: pod.ObjectMeta.Annotations,
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 将篡改后的 Patch 对象内容写入 response
	admissionReview := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: &admissionv1.AdmissionResponse{
			UID:     ar.Request.UID,
			Allowed: true,
			Patch:   patchBytes,
			PatchType: func() *admissionv1.PatchType {
				pt := admissionv1.PatchTypeJSONPatch
				return &pt
			}(),
		},
	}
	resp, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("资源变更成功.")
}
