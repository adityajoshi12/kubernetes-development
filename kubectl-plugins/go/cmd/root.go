package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubectl-decode/client"
)

type options struct {
	namespace string
}

func NewDecodeCMD() *cobra.Command {
	opt := options{}
	rootcmd := &cobra.Command{
		Use:     "decode",
		Aliases: []string{"decodes"},
		Short:   "decode secret",
		Long:    "decode kubernetes secret",
		Example: "kubectl decode my-secret -n default",
		RunE: func(cmd *cobra.Command, args []string) error {
			return decodeSecret(args[0], opt)
		},
		Version: "0.1",
	}
	rootcmd.Flags().StringVarP(&opt.namespace, "namespace", "n", "default", "secret namespace")
	return rootcmd
}

func decodeSecret(secretName string, opt options) error {

	client := client.K8Client()
	secret, err := client.CoreV1().Secrets(opt.namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for key, val := range secret.Data {
		fmt.Printf("%s:%s\n", key, string(val))
	}

	return nil
}
