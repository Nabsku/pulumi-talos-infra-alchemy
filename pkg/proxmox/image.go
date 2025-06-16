package proxmox

import (
	"fmt"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve/download"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/imagefactory"
)

func (p *Proxmox) DownloadTalosImage(ctx *pulumi.Context, factoryOutput *imagefactory.GetUrlsResultOutput) (*download.File, error) {
	if p.ComputeNodes == nil || len(*p.ComputeNodes) == 0 {
		return nil, fmt.Errorf("no compute nodes available to download the image")
	}

	nodeName := (*p.ComputeNodes)[0].Name()

	downloadedImage, err := download.NewFile(ctx, "talos-image", &download.FileArgs{
		Url:         factoryOutput.Urls().Iso(),
		ContentType: pulumi.String("iso"),
		FileName:    pulumi.String(fmt.Sprintf("talos-%s.iso", nodeName)),
		DatastoreId: pulumi.String("local"),
		NodeName:    pulumi.String(nodeName),
		Overwrite:   pulumi.Bool(true),
	}, pulumi.Provider(p.Provider))

	if err != nil {
		return nil, fmt.Errorf("failed to download Talos image: %w", err)
	}
	fmt.Printf("Downloaded Talos image to %s on node %s\n", downloadedImage.FileName, (*p.ComputeNodes)[0].Name())
	return downloadedImage, nil
}
