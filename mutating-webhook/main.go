package main

import (
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"

	"github.com/gin-gonic/gin"

	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"net/http"
)

func main() {
	r := gin.Default()
	r.GET("/healtz", func(c *gin.Context) {
		fmt.Println(c.Request.Method)
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/mutate", handleMutate)

	err := r.RunTLS(":8080", "/etc/certs/tls.crt", "/etc/certs/tls.key")
	if err != nil {
		panic(err)
	}

	log.Println("Starting server ...")

}

func handleMutate(c *gin.Context) {

	admissionReview := v1.AdmissionReview{}

	var err error
	if err = c.BindJSON(&admissionReview); err != nil {
		c.Abort()
		return
	}

	admissionReviewReq := admissionReview.Request

	log.Println("Incoming payload", admissionReviewReq)

	var pod *corev1.Pod

	if err = json.Unmarshal(admissionReviewReq.Object.Raw, &pod); err != nil {
		c.Abort()
		return
	}

	response := v1.AdmissionResponse{}

	patchType := v1.PatchTypeJSONPatch
	response.PatchType = &patchType
	response.UID = admissionReviewReq.UID

	if response.Patch, err = addResourceLimits(pod); err != nil {
		response.Allowed = false
		response.Result = &metav1.Status{
			Status: "Failed",
		}
		fmt.Println(err.Error())
	} else {
		response.Allowed = true
		response.Result = &metav1.Status{
			Status: "Success",
		}
	}

	admissionReview.Response = &response

	c.JSON(http.StatusOK, admissionReview)
}

func addResourceLimits(pod *corev1.Pod) ([]byte, error) {

	var patch []map[string]interface{}
	for i, container := range pod.Spec.Containers {
		if container.Resources.Limits == nil {
			patch = append(patch, map[string]interface{}{
				"op":   "add",
				"path": fmt.Sprintf("/spec/containers/%d/resources", i),
				"value": map[string]map[string]string{
					"requests": {
						"cpu":    "150m",
						"memory": "128Mi",
					},
					"limits": {
						"cpu":    "300m",
						"memory": "256Mi",
					},
				},
			},
			)
		}
	}

	return json.Marshal(patch)

}
