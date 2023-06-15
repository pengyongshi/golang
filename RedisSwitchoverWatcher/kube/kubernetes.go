package kube

import (
    "context"
    "fmt"
    "k8s.io/apimachinery/pkg/api/errors"
    v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "log"
    "strings"
    "time"
)

func CreateClient(masterUrl string,kubeconfig string) (*kubernetes.Clientset, error) {
    config, err := clientcmd.BuildConfigFromFlags(masterUrl, kubeconfig)
    if err != nil {
        return nil, fmt.Errorf("failed to build kubeconfig: %v", err)
    }


    kubectl, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %v", err)
    }
    return kubectl, err
}

func RestartDeployment(clientset *kubernetes.Clientset, namespace, deploymentName string) error {
    deploymentsClient := clientset.AppsV1().Deployments(namespace)

    // 获取Deployment对象
    deployment, err := deploymentsClient.Get(context.TODO(), deploymentName, v1.GetOptions{})
    if err != nil {
        if errors.IsNotFound(err) {
            return fmt.Errorf("deployment %s not found in namespace %s", deploymentName, namespace)
        }
        return fmt.Errorf("failed to get deployment: %v", err)
    }

    // 创建一个Deployment的深拷贝
    newDeployment := deployment.DeepCopy()

    // 修改Deployment的唯一标识符，触发重启
    newDeployment.Spec.Template.ObjectMeta.Annotations["restartTimestamp"] = fmt.Sprintf("%d", time.Now().Unix())

    // 更新Deployment对象
    _, err = deploymentsClient.Update(context.TODO(), newDeployment, v1.UpdateOptions{})
    if err != nil {
        return fmt.Errorf("failed to update deployment: %v", err)
    }
    return nil
}


func RolloutRestartNamespace(clientset *kubernetes.Clientset, namespace string) error {
    ctx := context.Background()
    deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, v1.ListOptions{})
    if err != nil {
        log.Fatalln(err)
    }
    for _, deployment := range deployments.Items {
        fmt.Println("Restarting Deployment:", deployment.Name)
        err := RestartDeployment(clientset, deployment.Namespace, deployment.Name)
        if err != nil {
            return fmt.Errorf("Failed to restart Deployment %s: %v\n", deployment.Name, err)
        } else {
            log.Printf("Deployment %s restarted successfully\n", deployment.Name)
        }
    }
    return nil
}

func RolloutRestartNamespaces(clientset *kubernetes.Clientset, Namespaces string) error {
    namespaces := strings.Split(Namespaces, ",")
    if len(namespaces) == 1 {
        RolloutRestartNamespace(clientset, namespaces[0])
    } else {
        for _,ns := range namespaces {
            RolloutRestartNamespace(clientset, ns)
        }
    }
    return nil
}


