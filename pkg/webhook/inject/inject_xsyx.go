package inject

import (
	"context"
	"strings"

	"github.com/chaos-mesh/chaos-mesh/pkg/webhook/config"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func injectCMTimeskew(pod *corev1.Pod, inj *config.InjectionConfig, cli client.Client) {
	log.Info("search injectCMTimeskew ", "pod", pod.String())
	cm := &corev1.ConfigMap{}
	if err := cli.Get(context.Background(), types.NamespacedName{
		Namespace: pod.Namespace,
		Name:      "timefake",
	}, cm); err != nil && strings.Contains(err.Error(), "not found") {
		log.Info("search timefake-cm error ", "msg", err.Error(), "pod", pod.Name)
		cli.Create(context.Background(), &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "timefake",
				Namespace: strings.Split(pod.Namespace, ",")[0],
			},
			Data: map[string]string{
				"timefake": "+0s",
			},
		})
	}
	inj.VolumeMounts = append(inj.VolumeMounts, corev1.VolumeMount{
		Name:      "timefake",
		MountPath: "/timefake",
	})
}

func appendAppRunArgs(pod *corev1.Pod, inj *config.InjectionConfig) {
	log.Info("----------------------------------------appendAppRunArgs debug-log----------------------------------------")
	log.Info("appendAppRunArgs", "len(pod.Spec.Containers)", len(pod.Spec.Containers))
	for i := range pod.Spec.Containers {
		log.Info("Containers", " container ", pod.Spec.Containers[i].Name)
	}
	for i := range inj.Environment {
		log.Info("Environment", " env ", inj.Environment[i].Name)
	}
	log.Info("----------------------------------------appendAppRunArgs debug-log----------------------------------------")
	for i := range pod.Spec.Containers {
		env := pod.Spec.Containers[i].Env
		for i := range env {
			if env[i].Name == "APP_RUN_ARGS" {
				env[i].Value = env[i].Value + " " + inj.Environment[0].Value
			}
			//有的应用没有APP_RUN_ARGS，只有JVM_OPTS_PREFIX
			if env[i].Name == "JVM_OPTS_PREFIX" {
				env[i].Value = env[i].Value + " " + inj.Environment[0].Value
			}
		}
	}
}