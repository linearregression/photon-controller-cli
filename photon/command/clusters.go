// Copyright (c) 2016 VMware, Inc. All Rights Reserved.
//
// This product is licensed to you under the Apache License, Version 2.0 (the "License").
// You may not use this product except in compliance with the License.
//
// This product may include a number of subcomponents with separate copyright notices and
// license terms. Your use of these subcomponents is subject to the terms and conditions
// of the subcomponent's license, as noted in the LICENSE file.

package command

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vmware/photon-controller-cli/photon/client"
	"github.com/vmware/photon-controller-cli/photon/utils"

	"github.com/codegangsta/cli"
	"github.com/vmware/photon-controller-go-sdk/photon"
)

// Creates a cli.Command for clusters
// Subcommands: create;   Usage: cluster create [<options>]
//              show;     Usage: cluster show <id>
//              list;     Usage: cluster list [<options>]
//              list_vms; Usage: cluster list_vms <id>
//              resize;   Usage: cluster resize <id> <new worker count> [<options>]
//              delete;   Usage: cluster delete <id>
func GetClusterCommand() cli.Command {
	command := cli.Command{
		Name:  "cluster",
		Usage: "Options for clusters",
		Subcommands: []cli.Command{
			{
				Name:  "create",
				Usage: "Create a new cluster",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "tenant, t",
						Usage: "Tenant name",
					},
					cli.StringFlag{
						Name:  "project, p",
						Usage: "Project name",
					},
					cli.StringFlag{
						Name:  "name, n",
						Usage: "Cluster name",
					},
					cli.StringFlag{
						Name:  "type, k",
						Usage: "Cluster type (accepted values are KUBERNETES, MESOS, or SWARM)",
					},
					cli.StringFlag{
						Name:  "vm_flavor, v",
						Usage: "VM flavor name",
					},
					cli.StringFlag{
						Name:  "disk_flavor, d",
						Usage: "Disk flavor name",
					},
					cli.StringFlag{
						Name:  "network_id, w",
						Usage: "VM network ID",
					},
					cli.IntFlag{
						Name:  "worker_count, c",
						Usage: "Worker count",
					},
					cli.StringFlag{
						Name:  "dns",
						Usage: "VM network DNS server IP address",
					},
					cli.StringFlag{
						Name:  "gateway",
						Usage: "VM network gateway IP address",
					},
					cli.StringFlag{
						Name:  "netmask",
						Usage: "VM network netmask",
					},
					cli.StringFlag{
						Name:  "master-ip",
						Usage: "Kubernetes master IP address (required for Kubernetes clusters)",
					},
					cli.StringFlag{
						Name:  "container-network",
						Usage: "CIDR representation of the container network, e.g. '10.2.0.0/16' (required for Kubernetes clusters)",
					},
					cli.StringFlag{
						Name:  "zookeeper1",
						Usage: "Static IP address with which to create Zookeeper node 1 (required for Mesos clusters)",
					},
					cli.StringFlag{
						Name:  "zookeeper2",
						Usage: "Static IP address with which to create Zookeeper node 2 (required for Mesos clusters)",
					},
					cli.StringFlag{
						Name:  "zookeeper3",
						Usage: "Static IP address with which to create Zookeeper node 3 (required for Mesos clusters)",
					},
					cli.StringFlag{
						Name:  "etcd1",
						Usage: "Static IP address with which to create etcd node 1 (required for Kubernetes and Swarm clusters)",
					},
					cli.StringFlag{
						Name:  "etcd2",
						Usage: "Static IP address with which to create etcd node 2 (required for Kubernetes and Swarm clusters)",
					},
					cli.StringFlag{
						Name:  "etcd3",
						Usage: "Static IP address with which to create etcd node 3 (required for Kubernetes and Swarm clusters)",
					},
					cli.StringFlag{
						Name:  "ssh-key",
						Usage: "The file path of the SSH key",
					},
					cli.IntFlag{
						Name:  "batchSize",
						Usage: "Batch size for expanding worker nodes",
					},
					cli.BoolFlag{
						Name:  "wait-for-ready",
						Usage: "Wait synchronously for the cluster to become ready and expanded fully",
					},
				},
				Action: func(c *cli.Context) {
					err := createCluster(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:  "show",
				Usage: "Show information about a cluster",
				Action: func(c *cli.Context) {
					err := showCluster(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:  "list",
				Usage: "List clusters",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "tenant, t",
						Usage: "Tenant name",
					},
					cli.StringFlag{
						Name:  "project, p",
						Usage: "Project name",
					},
					cli.BoolFlag{
						Name:  "summary, s",
						Usage: "Summary view",
					},
				},
				Action: func(c *cli.Context) {
					err := listClusters(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:  "list_vms",
				Usage: "List the VMs associated with a cluster",
				Action: func(c *cli.Context) {
					err := listVms(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:  "resize",
				Usage: "Resize a cluster",
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "wait-for-ready",
						Usage: "Wait synchronously for the cluster to become ready and expanded fully",
					},
				},
				Action: func(c *cli.Context) {
					err := resizeCluster(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:  "delete",
				Usage: "Delete a cluster",
				Action: func(c *cli.Context) {
					err := deleteCluster(c)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
		},
	}
	return command
}

// Sends a "create cluster" request to the API client based on the cli.Context
// Returns an error if one occurred
func createCluster(c *cli.Context, w io.Writer) error {
	err := checkArgNum(c.Args(), 0, "cluster create [<options>]")
	if err != nil {
		return err
	}

	tenantName := c.String("tenant")
	projectName := c.String("project")
	name := c.String("name")
	cluster_type := c.String("type")
	vm_flavor := c.String("vm_flavor")
	disk_flavor := c.String("disk_flavor")
	network_id := c.String("network_id")
	worker_count := c.Int("worker_count")
	dns := c.String("dns")
	gateway := c.String("gateway")
	netmask := c.String("netmask")
	master_ip := c.String("master-ip")
	container_network := c.String("container-network")
	zookeeper1 := c.String("zookeeper1")
	zookeeper2 := c.String("zookeeper2")
	zookeeper3 := c.String("zookeeper3")
	etcd1 := c.String("etcd1")
	etcd2 := c.String("etcd2")
	etcd3 := c.String("etcd3")
	batch_size := c.Int("batchSize")
	ssh_key := c.String("ssh-key")

	wait_for_ready := c.IsSet("wait-for-ready")

	const DEFAULT_WORKER_COUNT = 1

	client.Esxclient, err = client.GetClient(c.GlobalIsSet("non-interactive"))
	if err != nil {
		return err
	}

	tenant, err := verifyTenant(tenantName)
	if err != nil {
		return err
	}

	project, err := verifyProject(tenant.ID, projectName)
	if err != nil {
		return err
	}

	if !utils.IsNonInteractive(c) {
		name, err = askForInput("Cluster name: ", name)
		if err != nil {
			return err
		}
		cluster_type, err = askForInput("Cluster type: ", cluster_type)
		if err != nil {
			return err
		}
		if worker_count == 0 {
			worker_count_string, err := askForInput("Worker count: ", "")
			if err != nil {
				return err
			}
			worker_count, err = strconv.Atoi(worker_count_string)
			if err != nil {
				return fmt.Errorf("Please supply a valid worker count")
			}
		}
	}

	if len(name) == 0 || len(cluster_type) == 0 {
		return fmt.Errorf("Provide a valid cluster name and type")
	}

	if worker_count == 0 {
		worker_count = DEFAULT_WORKER_COUNT
	}

	if !utils.IsNonInteractive(c) {
		dns, err = askForInput("Cluster DNS server: ", dns)
		if err != nil {
			return err
		}
		gateway, err = askForInput("Cluster network gateway: ", gateway)
		if err != nil {
			return err
		}
		netmask, err = askForInput("Cluster network netmask: ", netmask)
		if err != nil {
			return err
		}
		ssh_key, err = askForInput("Cluster ssh key file path (leave blank for none): ", ssh_key)
		if err != nil {
			return err
		}
	}

	if len(dns) == 0 || len(gateway) == 0 || len(netmask) == 0 {
		return fmt.Errorf("Provide a valid DNS, gateway, and netmask")
	}

	extended_properties := make(map[string]string)
	extended_properties[photon.ExtendedPropertyDNS] = dns
	extended_properties[photon.ExtendedPropertyGateway] = gateway
	extended_properties[photon.ExtendedPropertyNetMask] = netmask
	if len(ssh_key) != 0 {
		ssh_key_content, err := readSSHKey(ssh_key)
		if err == nil {
			extended_properties[photon.ExtendedPropertySSHKey] = ssh_key_content
		} else {
			return err
		}
	}

	cluster_type = strings.ToUpper(cluster_type)
	switch cluster_type {
	case "KUBERNETES":
		if !utils.IsNonInteractive(c) {
			master_ip, err = askForInput("Kubernetes master static IP address: ", master_ip)
			if err != nil {
				return err
			}
			container_network, err = askForInput("Kubernetes worker network ID: ", container_network)
			if err != nil {
				return err
			}
			etcd1, err = askForInput("etcd server 1 static IP address: ", etcd1)
			if err != nil {
				return err
			}
			etcd2, err = askForInput("etcd server 2 static IP address (leave blank for none): ", etcd2)
			if err != nil {
				return err
			}
			if len(etcd2) != 0 {
				etcd3, err = askForInput("etcd server 3 static IP address (leave blank for none): ", etcd3)
				if err != nil {
					return err
				}
			}
		}

		extended_properties[photon.ExtendedPropertyMasterIP] = master_ip
		extended_properties[photon.ExtendedPropertyContainerNetwork] = container_network
		extended_properties[photon.ExtendedPropertyETCDIP1] = etcd1
		if len(etcd2) != 0 {
			extended_properties[photon.ExtendedPropertyETCDIP2] = etcd2
			if len(etcd3) != 0 {
				extended_properties[photon.ExtendedPropertyETCDIP3] = etcd3
			}
		}
	case "MESOS":
		if !utils.IsNonInteractive(c) {
			zookeeper1, err = askForInput("Zookeeper server 1 static IP address: ", zookeeper1)
			if err != nil {
				return err
			}
			zookeeper2, err = askForInput("Zookeeper server 2 static IP address (leave blank for none): ", zookeeper2)
			if err != nil {
				return err
			}
			if len(zookeeper2) != 0 {
				zookeeper3, err = askForInput("Zookeeper server 3 static IP address (leave blank for none): ", zookeeper3)
				if err != nil {
					return err
				}
			}
		}

		extended_properties[photon.ExtendedPropertyZookeeperIP1] = zookeeper1
		if len(zookeeper2) != 0 {
			extended_properties[photon.ExtendedPropertyZookeeperIP2] = zookeeper2
			if len(zookeeper3) != 0 {
				extended_properties[photon.ExtendedPropertyZookeeperIP3] = zookeeper3
			}
		}
	case "SWARM":
		if !utils.IsNonInteractive(c) {
			etcd1, err = askForInput("etcd server 1 static IP address: ", etcd1)
			if err != nil {
				return err
			}
			etcd2, err = askForInput("etcd server 2 static IP address (leave blank for none): ", etcd2)
			if err != nil {
				return err
			}
			if len(etcd2) != 0 {
				etcd3, err = askForInput("etcd server 3 static IP address (leave blank for none): ", etcd3)
				if err != nil {
					return err
				}
			}
		}

		extended_properties[photon.ExtendedPropertyETCDIP1] = etcd1
		if len(etcd2) != 0 {
			extended_properties[photon.ExtendedPropertyETCDIP2] = etcd2
			if len(etcd3) != 0 {
				extended_properties[photon.ExtendedPropertyETCDIP3] = etcd3
			}
		}
	default:
		return fmt.Errorf("Unsupported cluster type: %s", cluster_type)
	}

	clusterSpec := photon.ClusterCreateSpec{}
	clusterSpec.Name = name
	clusterSpec.Type = cluster_type
	clusterSpec.VMFlavor = vm_flavor
	clusterSpec.DiskFlavor = disk_flavor
	clusterSpec.NetworkID = network_id
	clusterSpec.WorkerCount = worker_count
	clusterSpec.BatchSizeWorker = batch_size
	clusterSpec.ExtendedProperties = extended_properties

	if !utils.IsNonInteractive(c) {
		fmt.Printf("\n")
		fmt.Printf("Creating cluster: %s (%s)\n", clusterSpec.Name, clusterSpec.Type)
		if len(clusterSpec.VMFlavor) != 0 {
			fmt.Printf("  VM flavor: %s\n", clusterSpec.VMFlavor)
		}
		if len(clusterSpec.DiskFlavor) != 0 {
			fmt.Printf("  Disk flavor: %s\n", clusterSpec.DiskFlavor)
		}
		fmt.Printf("  Worker count: %d\n", clusterSpec.WorkerCount)
		if clusterSpec.BatchSizeWorker != 0 {
			fmt.Printf("  Batch size: %d\n", clusterSpec.BatchSizeWorker)
		}
		fmt.Printf("\n")
	}

	if confirmed(utils.IsNonInteractive(c)) {
		createTask, err := client.Esxclient.Projects.CreateCluster(project.ID, &clusterSpec)
		if err != nil {
			return err
		}

		_, err = waitOnTaskOperation(createTask.ID, c)
		if err != nil {
			return err
		}

		if wait_for_ready {
			if !utils.NeedsFormatting(c) {
				fmt.Printf("Waiting for cluster %s to become ready\n", createTask.Entity.ID)
			}
			cluster, err := waitForCluster(createTask.Entity.ID)
			if err != nil {
				return err
			}

			if utils.NeedsFormatting(c) {
				utils.FormatObject(cluster, w, c)
			} else {
				fmt.Printf("Cluster %s is ready\n", cluster.ID)
			}

		} else {
			fmt.Println("Note: the cluster has been created with minimal resources. You can use the cluster now.")
			fmt.Println("A background task is running to gradually expand the cluster to its target capacity.")
			fmt.Printf("You can run 'cluster show %s' to see the state of the cluster.\n", createTask.Entity.ID)
		}
	} else {
		fmt.Println("Cancelled")
	}

	return nil
}

// Sends a "show cluster" request to the API client based on the cli.Context
// Returns an error if one occurred
func showCluster(c *cli.Context, w io.Writer) error {
	err := checkArgNum(c.Args(), 1, "cluster show <id>")
	if err != nil {
		return err
	}
	id := c.Args().First()

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	cluster, err := client.Esxclient.Clusters.Get(id)
	if err != nil {
		return err
	}

	vms, err := client.Esxclient.Clusters.GetVMs(id)
	if err != nil {
		return err
	}

	var master_vms []photon.VM
	for _, vm := range vms.Items {
		for _, tag := range vm.Tags {
			if strings.Count(tag, ":") == 2 && !strings.Contains(strings.ToLower(tag), "worker") {
				master_vms = append(master_vms, vm)
				break
			}
		}
	}

	if c.GlobalIsSet("non-interactive") {
		extendedProperties := strings.Trim(strings.TrimLeft(fmt.Sprint(cluster.ExtendedProperties), "map"), "[]")
		fmt.Printf("%s\t%s\t%s\t%s\t%d\t%s\n", cluster.ID, cluster.Name, cluster.State, cluster.Type,
			cluster.WorkerCount, extendedProperties)
	} else if utils.NeedsFormatting(c) {
		utils.FormatObject(cluster, w, c)
	} else {
		fmt.Println("Cluster ID:            ", cluster.ID)
		fmt.Println("  Name:                ", cluster.Name)
		fmt.Println("  State:               ", cluster.State)
		fmt.Println("  Type:                ", cluster.Type)
		fmt.Println("  Worker count:        ", cluster.WorkerCount)
		fmt.Println("  Extended Properties: ", cluster.ExtendedProperties)
		fmt.Println()
	}

	err = printClusterVMs(master_vms, c.GlobalIsSet("non-interactive"))
	if err != nil {
		return err
	}

	return nil
}

// Sends a "list clusters" request to the API client based on the cli.Context
// Returns an error if one occurred
func listClusters(c *cli.Context, w io.Writer) error {
	err := checkArgNum(c.Args(), 0, "cluster list [<options>]")
	if err != nil {
		return err
	}

	tenantName := c.String("tenant")
	projectName := c.String("project")
	summaryView := c.IsSet("summary")

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	tenant, err := verifyTenant(tenantName)
	if err != nil {
		return err
	}

	project, err := verifyProject(tenant.ID, projectName)
	if err != nil {
		return err
	}

	clusterList, err := client.Esxclient.Projects.GetClusters(project.ID)
	if err != nil {
		return err
	}

	err = printClusterList(clusterList.Items, w, c, summaryView)
	if err != nil {
		return err
	}

	return nil
}

// Sends a "list VMs for cluster" request to the API client based on the cli.Context
// Returns an error if one occurred
func listVms(c *cli.Context, w io.Writer) error {
	err := checkArgNum(c.Args(), 1, "cluster list_vms <id>")
	if err != nil {
		return err
	}
	cluster_id := c.Args().First()

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	vms, err := client.Esxclient.Clusters.GetVMs(cluster_id)
	if err != nil {
		return err
	}

	err = printVMList(vms.Items, w, c, false)
	if err != nil {
		return err
	}

	return nil
}

// Sends a "resize cluster" request to the API client based on the cli.Context
// Returns an error if one occurred
func resizeCluster(c *cli.Context, w io.Writer) error {
	err := checkArgNum(c.Args(), 2, "cluster resize <id> <new worker count> [<options>]")
	if err != nil {
		return err
	}

	cluster_id := c.Args()[0]
	worker_count_string := c.Args()[1]
	worker_count, err := strconv.Atoi(worker_count_string)
	wait_for_ready := c.IsSet("wait-for-ready")

	if len(cluster_id) == 0 || err != nil || worker_count <= 0 {
		return fmt.Errorf("Provide a valid cluster ID and worker count")
	}

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	if !utils.IsNonInteractive(c) {
		fmt.Printf("\nResizing cluster %s to worker count %d\n", cluster_id, worker_count)
	}

	if confirmed(utils.IsNonInteractive(c)) {
		resizeSpec := photon.ClusterResizeOperation{}
		resizeSpec.NewWorkerCount = worker_count
		resizeTask, err := client.Esxclient.Clusters.Resize(cluster_id, &resizeSpec)
		if err != nil {
			return err
		}

		_, err = waitOnTaskOperation(resizeTask.ID, c)
		if err != nil {
			return err
		}

		if wait_for_ready {
			cluster, err := waitForCluster(cluster_id)
			if err != nil {
				return err
			}
			if utils.NeedsFormatting(c) {
				utils.FormatObject(cluster, w, c)
			} else {
				fmt.Printf("Cluster %s is ready\n", cluster.ID)
			}
		} else {
			fmt.Println("Note: A background task is running to gradually resize the cluster to its target capacity.")
			fmt.Printf("You may continue to use the cluster. You can run 'cluster show %s'\n", resizeTask.Entity.ID)
			fmt.Println("to see the state of the cluster. If the resize operation is still in progress, the cluster state")
			fmt.Println("will show as RESIZING. Once the cluster is resized, the cluster state will show as READY.")
		}
	} else {
		fmt.Println("Cancelled")
	}

	return nil
}

// Sends a "delete cluster" request to the API client based on the cli.Context
// Returns an error if one occurred
func deleteCluster(c *cli.Context) error {
	err := checkArgNum(c.Args(), 1, "cluster delete <id>")
	if err != nil {
		return nil
	}

	cluster_id := c.Args().First()

	if len(cluster_id) == 0 {
		return fmt.Errorf("Please provide a valid cluster ID")
	}

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	if !utils.IsNonInteractive(c) {
		fmt.Printf("\nDeleting cluster %s\n", cluster_id)
	}

	if confirmed(utils.IsNonInteractive(c)) {
		deleteTask, err := client.Esxclient.Clusters.Delete(cluster_id)
		if err != nil {
			return err
		}

		_, err = waitOnTaskOperation(deleteTask.ID, c)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("Cancelled")
	}

	return nil
}

// Helper routine which waits for a cluster to enter the READY state.
func waitForCluster(id string) (cluster *photon.Cluster, err error) {
	start := time.Now()
	numErr := 0

	taskPollTimeout := 60 * time.Minute
	taskPollDelay := 2 * time.Second
	taskRetryCount := 3

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		displayTaskProgress(start)
	}()

	for time.Since(start) < taskPollTimeout {
		cluster, err = client.Esxclient.Clusters.Get(id)
		if err != nil {
			numErr++
			if numErr > taskRetryCount {
				endAnimation = true
				wg.Wait()
				return
			}
		}
		switch strings.ToUpper(cluster.State) {
		case "ERROR":
			endAnimation = true
			wg.Wait()
			err = fmt.Errorf("Cluster %s entered ERROR state", id)
			return
		case "READY":
			endAnimation = true
			wg.Wait()
			return
		}

		time.Sleep(taskPollDelay)
	}

	endAnimation = true
	wg.Wait()
	err = fmt.Errorf("Timed out while waiting for cluster to enter READY state")
	return
}

// This is a helper function to for reading the ssh key from a file.
func readSSHKey(filename string) (result string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer func() {
		e := file.Close()
		if e != nil {
			err = e
		}
	}()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	// Read just the first line because the key file should only by one line long.
	scanner.Scan()
	keystring := scanner.Text()
	keystring = strings.TrimSpace(keystring)
	if err := scanner.Err(); err != nil {
		return "", err
	}
	err = validateSSHKey(keystring)
	if err != nil {
		return "", err
	}
	return keystring, nil
}

// This is a helper function to validate that a key is a valid ssh key
func validateSSHKey(key string) error {
	if len(key) == 0 {
		return fmt.Errorf("The ssh-key file provided has no content")
	}
	// Other validation test can go here if desired in the future
	return nil
}
