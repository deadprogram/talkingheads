package actor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hybridgroup/yzma/pkg/download"
)

func EnsureInstall(libPath string, processor string, updateInstall bool) error {
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		if err := os.Mkdir(libPath, 0755); err != nil {
			fmt.Println("failed to create llama.cpp library directory:", err.Error())
			return err
		}
	}

	if !download.AlreadyInstalled(libPath) || updateInstall {
		version, err := download.LlamaLatestVersion()
		if err != nil {
			fmt.Println("could not obtain latest version:", err.Error())
			return err
		}

		if processor == "" {
			processor = "cpu"
			if cudaInstalled, cudaVersion := download.HasCUDA(); cudaInstalled {
				fmt.Printf("CUDA detected (version %s), using CUDA build\n", cudaVersion)
				processor = "cuda"
			}
		}

		fmt.Println("installing llama.cpp version", version, "to", libPath)
		if err := download.GetWithContext(context.Background(), runtime.GOARCH, "trixie", processor, version, libPath, download.ProgressTracker); err != nil {
			fmt.Println("failed to download llama.cpp:", err.Error())
			return err
		}
		fmt.Println("yzma installed successfully.")
	}

	return nil
}

func EnsureModel(modelUrl string, modelsPath string) error {
	if len(modelUrl) == 0 {
		return fmt.Errorf("model URL is required")
	}

	modelName := filepath.Base(modelUrl)
	modelFilePath := filepath.Join(modelsPath, modelName)

	if _, err := os.Stat(modelFilePath); os.IsNotExist(err) {
		fmt.Println("installing model from", modelUrl, "to", modelFilePath)
		if err := download.GetModelWithContext(context.Background(), modelUrl, modelsPath, download.ProgressTracker); err != nil {
			fmt.Println("failed to download model:", err.Error())
			return err
		}

		fmt.Println("model downloaded successfully.")
	} else {
		fmt.Println("model already exists, skipping download.")
	}

	return nil
}
