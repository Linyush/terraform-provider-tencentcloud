package tencentcloud

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
	"github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud/connectivity"
	"github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud/internal/helper"
	"github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud/ratelimit"
)

type ClusterBasicSetting struct {
	ClusterId          string
	ClusterOs          string
	ClusterOsType      string
	ClusterVersion     string
	ClusterName        string
	ClusterDescription string
	VpcId              string
	ProjectId          int64
	ClusterNodeNum     int64
	ClusterStatus      string
	Tags               map[string]string
}

type ClusterAdvancedSettings struct {
	Ipvs                    bool
	AsEnabled               bool
	EnableCustomizedPodCIDR bool
	BasePodNumber           int64
	ContainerRuntime        string
	NodeNameType            string
	ExtraArgs               ClusterExtraArgs
	NetworkType             string
	IsNonStaticIpMode       bool
	DeletionProtection      bool
	KubeProxyMode           string
}

type ClusterExtraArgs struct {
	KubeAPIServer         []string
	KubeControllerManager []string
	KubeScheduler         []string
}

type RunInstancesForNode struct {
	Master []string
	Work   []string
}

type InstanceAdvancedSettings struct {
	MountTarget     string
	DockerGraphPath string
	UserScript      string
	Unschedulable   int64
	DesiredPodNum   int64
	Labels          []*tke.Label
	ExtraArgs       tke.InstanceExtraArgs
}

type ClusterCidrSettings struct {
	ClusterCidr               string
	IgnoreClusterCidrConflict bool
	MaxNodePodNum             int64
	MaxClusterServiceNum      int64
	ServiceCIDR               string
	EniSubnetIds              []string
	ClaimExpiredSeconds       int64
}

type ClusterInfo struct {
	ClusterBasicSetting
	ClusterCidrSettings
	ClusterAdvancedSettings

	DeployType string
}

type InstanceInfo struct {
	InstanceId               string
	InstanceRole             string
	InstanceState            string
	FailedReason             string
	InstanceAdvancedSettings *tke.InstanceAdvancedSettings
	LanIp                    string
}

type TkeService struct {
	client *connectivity.TencentCloudClient
}

func (me *TkeService) DescribeClusterInstances(ctx context.Context, id string) (masters []InstanceInfo, workers []InstanceInfo, errRet error) {
	logId := getLogId(ctx)
	request := tke.NewDescribeClusterInstancesRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	request.ClusterId = &id
	masters = make([]InstanceInfo, 0, 100)
	workers = make([]InstanceInfo, 0, 100)
	var offset int64 = 0
	var limit int64 = 20
	var has = map[string]bool{}
	var total int64 = -1

getMoreData:
	if total >= 0 && offset >= total {
		return
	}
	request.Limit = &limit
	request.Offset = &offset
	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().DescribeClusterInstances(request)
	if err != nil {
		errRet = err
		return
	}
	if total < 0 {
		total = int64(*response.Response.TotalCount)
	}

	if len(response.Response.InstanceSet) > 0 {
		offset += limit
	} else {
		// get empty set, we're done
		return
	}

	for _, item := range response.Response.InstanceSet {
		if has[*item.InstanceId] {
			errRet = fmt.Errorf("get repeated instance_id[%s] when doing DescribeClusterInstances", *item.InstanceId)
			return
		}
		has[*item.InstanceId] = true
		instanceInfo := InstanceInfo{
			InstanceId:               *item.InstanceId,
			InstanceRole:             *item.InstanceRole,
			InstanceState:            *item.InstanceState,
			FailedReason:             *item.FailedReason,
			InstanceAdvancedSettings: item.InstanceAdvancedSettings,
		}
		if item.LanIP != nil {
			instanceInfo.LanIp = *item.LanIP
		}
		if instanceInfo.InstanceRole == TKE_ROLE_WORKER {
			workers = append(workers, instanceInfo)
		} else {
			masters = append(masters, instanceInfo)
		}
	}
	goto getMoreData

}

func (me *TkeService) DescribeClusters(ctx context.Context, id string, name string) (clusterInfos []ClusterInfo, errRet error) {

	logId := getLogId(ctx)
	request := tke.NewDescribeClustersRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	if id != "" && name != "" {
		errRet = fmt.Errorf("cluster_id, cluster_name only one can be set one")
		return
	}

	if id != "" {
		request.ClusterIds = []*string{&id}
	}

	if name != "" {
		filter := &tke.Filter{
			Name:   helper.String("ClusterName"),
			Values: []*string{&name},
		}
		request.Filters = []*tke.Filter{filter}
	}

	response, err := me.client.UseTkeClient().DescribeClusters(request)

	if err != nil {
		errRet = err
		return
	}

	lenClusters := len(response.Response.Clusters)

	if lenClusters == 0 {
		return
	}
	clusterInfos = make([]ClusterInfo, 0, lenClusters)

	for index := range response.Response.Clusters {
		cluster := response.Response.Clusters[index]
		var clusterInfo ClusterInfo

		clusterInfo.ClusterId = *cluster.ClusterId
		clusterInfo.ClusterOs = *cluster.ClusterOs
		clusterInfo.ClusterVersion = *cluster.ClusterVersion
		clusterInfo.ClusterDescription = *cluster.ClusterDescription
		clusterInfo.ClusterName = *cluster.ClusterName
		clusterInfo.ClusterStatus = *cluster.ClusterStatus

		clusterInfo.ProjectId = int64(*cluster.ProjectId)
		clusterInfo.VpcId = *cluster.ClusterNetworkSettings.VpcId
		clusterInfo.ClusterNodeNum = int64(*cluster.ClusterNodeNum)

		clusterInfo.IgnoreClusterCidrConflict = *cluster.ClusterNetworkSettings.IgnoreClusterCIDRConflict
		clusterInfo.ClusterCidr = *cluster.ClusterNetworkSettings.ClusterCIDR
		clusterInfo.MaxClusterServiceNum = int64(*cluster.ClusterNetworkSettings.MaxClusterServiceNum)

		clusterInfo.MaxNodePodNum = int64(*cluster.ClusterNetworkSettings.MaxNodePodNum)
		clusterInfo.DeployType = strings.ToUpper(*cluster.ClusterType)
		clusterInfo.Ipvs = *cluster.ClusterNetworkSettings.Ipvs

		if len(cluster.TagSpecification) > 0 {
			clusterInfo.Tags = make(map[string]string)
			for _, tag := range cluster.TagSpecification[0].Tags {
				clusterInfo.Tags[*tag.Key] = *tag.Value
			}
		}

		clusterInfos = append(clusterInfos, clusterInfo)
	}
	return
}

func (me *TkeService) DescribeCluster(ctx context.Context, id string) (
	clusterInfo ClusterInfo,
	has bool,
	errRet error,
) {

	logId := getLogId(ctx)
	request := tke.NewDescribeClustersRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	request.ClusterIds = []*string{&id}

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().DescribeClusters(request)

	if err != nil {
		errRet = err
		return
	}

	if len(response.Response.Clusters) == 0 {
		return
	}

	has = true
	cluster := response.Response.Clusters[0]
	clusterInfo.ClusterId = *cluster.ClusterId
	clusterInfo.ClusterOs = *cluster.ClusterOs
	clusterInfo.ClusterVersion = *cluster.ClusterVersion
	clusterInfo.ClusterDescription = *cluster.ClusterDescription
	clusterInfo.ClusterName = *cluster.ClusterName
	clusterInfo.ClusterStatus = *cluster.ClusterStatus

	clusterInfo.ProjectId = int64(*cluster.ProjectId)
	clusterInfo.VpcId = *cluster.ClusterNetworkSettings.VpcId
	clusterInfo.ClusterNodeNum = int64(*cluster.ClusterNodeNum)

	clusterInfo.IgnoreClusterCidrConflict = *cluster.ClusterNetworkSettings.IgnoreClusterCIDRConflict
	clusterInfo.ClusterCidr = *cluster.ClusterNetworkSettings.ClusterCIDR
	clusterInfo.MaxClusterServiceNum = int64(*cluster.ClusterNetworkSettings.MaxClusterServiceNum)

	clusterInfo.MaxNodePodNum = int64(*cluster.ClusterNetworkSettings.MaxNodePodNum)
	clusterInfo.DeployType = strings.ToUpper(*cluster.ClusterType)
	clusterInfo.Ipvs = *cluster.ClusterNetworkSettings.Ipvs

	if len(cluster.TagSpecification) > 0 {
		clusterInfo.Tags = make(map[string]string)
		for _, tag := range cluster.TagSpecification[0].Tags {
			clusterInfo.Tags[*tag.Key] = *tag.Value
		}
	}

	return
}

func (me *TkeService) DescribeClusterConfig(ctx context.Context, id string) (config string, errRet error) {

	logId := getLogId(ctx)
	request := tke.NewDescribeClusterKubeconfigRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	request.ClusterId = &id

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().DescribeClusterKubeconfig(request)

	if err != nil {
		errRet = err
		return
	}

	if response == nil || response.Response == nil {
		return
	}

	config = *response.Response.Kubeconfig
	return
}

func (me *TkeService) GetUpgradeInstanceResult(ctx context.Context, id string) (
	done bool,
	errRet error,
) {

	logId := getLogId(ctx)
	request := tke.NewGetUpgradeInstanceProgressRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	request.ClusterId = &id

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().GetUpgradeInstanceProgress(request)

	if err != nil {
		errRet = err
		return
	}

	lifeState := *response.Response.LifeState

	// all instances success, lifeState=done
	if lifeState == "done" {
		return true, nil
	} else if lifeState != "process" {
		return false, fmt.Errorf("upgrade instances failed, tke response lifeState is:%s", lifeState)
	}

	// parent lifeState=process, check whether all instances in processing.
	for _, inst := range response.Response.Instances {
		if *inst.LifeState == "done" || *inst.LifeState == "pending" {
			continue
		}
		if *inst.LifeState != "process" {
			return false, fmt.Errorf("upgrade instances failed, "+
				"instanceId:%s, lifeState is:%s", *inst.InstanceID, *inst.LifeState)
		}
		// instance lifeState=process, check whether failed or not.
		for _, detail := range inst.Detail {
			if *detail.LifeState == "failed" {
				return false, fmt.Errorf("upgrade instances failed, "+
					"instanceId:%s, detail.lifeState is:%s", *inst.InstanceID, *detail.LifeState)
			}
		}
	}

	return
}

func (me *TkeService) CreateCluster(ctx context.Context,
	basic ClusterBasicSetting,
	advanced ClusterAdvancedSettings,
	cvms RunInstancesForNode,
	iAdvanced InstanceAdvancedSettings,
	cidrSetting ClusterCidrSettings,
	tags map[string]string,
	existedInstance []*tke.ExistedInstancesForNode,
	overrideSettings *OverrideSettings,
) (id string, errRet error) {

	logId := getLogId(ctx)
	request := tke.NewCreateClusterRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	request.ClusterBasicSettings = &tke.ClusterBasicSettings{}
	request.ClusterBasicSettings.ClusterOs = &basic.ClusterOs
	request.ClusterBasicSettings.ClusterVersion = &basic.ClusterVersion
	request.ClusterBasicSettings.ProjectId = &basic.ProjectId
	request.ClusterBasicSettings.VpcId = &basic.VpcId
	request.ClusterBasicSettings.ClusterDescription = &basic.ClusterDescription
	request.ClusterBasicSettings.ClusterName = &basic.ClusterName
	request.ClusterBasicSettings.OsCustomizeType = &basic.ClusterOsType
	for k, v := range tags {
		if len(request.ClusterBasicSettings.TagSpecification) == 0 {
			request.ClusterBasicSettings.TagSpecification = []*tke.TagSpecification{{
				ResourceType: helper.String("cluster"),
			}}
		}

		request.ClusterBasicSettings.TagSpecification[0].Tags = append(request.ClusterBasicSettings.TagSpecification[0].Tags, &tke.Tag{
			Key:   helper.String(k),
			Value: helper.String(v),
		})
	}

	request.ClusterAdvancedSettings = &tke.ClusterAdvancedSettings{}
	request.ClusterAdvancedSettings.IPVS = &advanced.Ipvs
	request.ClusterAdvancedSettings.AsEnabled = &advanced.AsEnabled
	request.ClusterAdvancedSettings.ContainerRuntime = &advanced.ContainerRuntime
	request.ClusterAdvancedSettings.NodeNameType = &advanced.NodeNameType
	request.ClusterAdvancedSettings.EnableCustomizedPodCIDR = &advanced.EnableCustomizedPodCIDR
	request.ClusterAdvancedSettings.BasePodNumber = &advanced.BasePodNumber
	request.ClusterAdvancedSettings.ExtraArgs = &tke.ClusterExtraArgs{
		KubeAPIServer:         common.StringPtrs(advanced.ExtraArgs.KubeAPIServer),
		KubeControllerManager: common.StringPtrs(advanced.ExtraArgs.KubeControllerManager),
		KubeScheduler:         common.StringPtrs(advanced.ExtraArgs.KubeScheduler),
	}
	request.ClusterAdvancedSettings.NetworkType = &advanced.NetworkType
	request.ClusterAdvancedSettings.IsNonStaticIpMode = &advanced.IsNonStaticIpMode
	request.ClusterAdvancedSettings.DeletionProtection = &advanced.DeletionProtection
	request.ClusterAdvancedSettings.KubeProxyMode = &advanced.KubeProxyMode

	request.InstanceAdvancedSettings = &tke.InstanceAdvancedSettings{}
	request.InstanceAdvancedSettings.MountTarget = &iAdvanced.MountTarget
	request.InstanceAdvancedSettings.DockerGraphPath = &iAdvanced.DockerGraphPath
	request.InstanceAdvancedSettings.UserScript = &iAdvanced.UserScript
	request.InstanceAdvancedSettings.Unschedulable = &iAdvanced.Unschedulable
	request.InstanceAdvancedSettings.DesiredPodNumber = &iAdvanced.DesiredPodNum
	if len(iAdvanced.ExtraArgs.Kubelet) > 0 {
		request.InstanceAdvancedSettings.ExtraArgs = &iAdvanced.ExtraArgs
	}

	if len(iAdvanced.Labels) > 0 {
		request.InstanceAdvancedSettings.Labels = iAdvanced.Labels
	}

	if len(overrideSettings.Master)+len(overrideSettings.Work) > 0 &&
		len(overrideSettings.Master)+len(overrideSettings.Work) != (len(cvms.Master)+len(cvms.Work)) {
		return "", fmt.Errorf("len(overrideSettings) != (len(cvms.Master)+len(cvms.Work))")
	}

	request.RunInstancesForNode = []*tke.RunInstancesForNode{}

	if len(cvms.Master) != 0 {

		var node tke.RunInstancesForNode
		node.NodeRole = helper.String(TKE_ROLE_MASTER_ETCD)
		node.RunInstancesPara = []*string{}
		request.ClusterType = helper.String(TKE_DEPLOY_TYPE_INDEPENDENT)
		for v := range cvms.Master {
			node.RunInstancesPara = append(node.RunInstancesPara, &cvms.Master[v])
			if len(overrideSettings.Master) != 0 {
				node.InstanceAdvancedSettingsOverrides = append(node.InstanceAdvancedSettingsOverrides, &overrideSettings.Master[v])
			}
		}
		request.RunInstancesForNode = append(request.RunInstancesForNode, &node)

	} else {
		request.ClusterType = helper.String(TKE_DEPLOY_TYPE_MANAGED)
	}

	if len(cvms.Work) != 0 {
		var node tke.RunInstancesForNode
		node.NodeRole = helper.String(TKE_ROLE_WORKER)
		node.RunInstancesPara = []*string{}
		for v := range cvms.Work {
			node.RunInstancesPara = append(node.RunInstancesPara, &cvms.Work[v])
			if len(overrideSettings.Work) != 0 {
				node.InstanceAdvancedSettingsOverrides = append(node.InstanceAdvancedSettingsOverrides, &overrideSettings.Work[v])
			}
		}
		request.RunInstancesForNode = append(request.RunInstancesForNode, &node)
	}

	request.ClusterCIDRSettings = &tke.ClusterCIDRSettings{}

	maxNodePodNum := uint64(cidrSetting.MaxNodePodNum)
	request.ClusterCIDRSettings.MaxNodePodNum = &maxNodePodNum

	maxClusterServiceNum := uint64(cidrSetting.MaxClusterServiceNum)
	request.ClusterCIDRSettings.MaxClusterServiceNum = &maxClusterServiceNum
	request.ClusterCIDRSettings.ClusterCIDR = &cidrSetting.ClusterCidr
	request.ClusterCIDRSettings.IgnoreClusterCIDRConflict = &cidrSetting.IgnoreClusterCidrConflict
	request.ClusterCIDRSettings.ServiceCIDR = &cidrSetting.ServiceCIDR
	request.ClusterCIDRSettings.EniSubnetIds = common.StringPtrs(cidrSetting.EniSubnetIds)
	request.ClusterCIDRSettings.ClaimExpiredSeconds = &cidrSetting.ClaimExpiredSeconds

	if len(existedInstance) > 0 {
		request.ExistedInstancesForNode = existedInstance
	}

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().CreateCluster(request)

	if err != nil {
		errRet = err
		return
	}

	id = *response.Response.ClusterId
	return
}

func (me *TkeService) CreateClusterInstances(ctx context.Context,
	id string, runInstancePara string,
	iAdvanced tke.InstanceAdvancedSettings) (instanceIds []string, errRet error) {
	logId := getLogId(ctx)
	request := tke.NewCreateClusterInstancesRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.RunInstancePara = &runInstancePara

	request.InstanceAdvancedSettings = &iAdvanced

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().CreateClusterInstances(request)

	if err != nil {
		errRet = err
		return
	}

	if response == nil || response.Response == nil {
		errRet = fmt.Errorf("CreateClusterInstances return nil response")
		return
	}

	instanceIds = make([]string, 0, len(response.Response.InstanceIdSet))

	for _, v := range response.Response.InstanceIdSet {

		instanceIds = append(instanceIds, *v)
	}
	return
}

/*
	if cluster is creating, return error:TencentCloudSDKError] Code=InternalError.ClusterState
*/
func (me *TkeService) DeleteClusterInstances(ctx context.Context, id string, instanceIds []string) (errRet error) {
	logId := getLogId(ctx)
	request := tke.NewDeleteClusterInstancesRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.InstanceIds = make([]*string, 0, len(instanceIds))

	for index := range instanceIds {
		request.InstanceIds = append(request.InstanceIds, &instanceIds[index])
	}

	request.InstanceDeleteMode = helper.String("terminate")
	ratelimit.Check(request.GetAction())
	_, err := me.client.UseTkeClient().DeleteClusterInstances(request)
	return err
}

func (me *TkeService) DeleteCluster(ctx context.Context, id string) (errRet error) {

	logId := getLogId(ctx)
	request := tke.NewDeleteClusterRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.InstanceDeleteMode = helper.String("terminate")

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseTkeClient().DeleteCluster(request)

	return err
}

func (me *TkeService) DescribeClusterSecurity(ctx context.Context, id string) (ret *tke.DescribeClusterSecurityResponse, errRet error) {

	logId := getLogId(ctx)
	request := tke.NewDescribeClusterSecurityRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()
	request.ClusterId = &id

	return me.client.UseTkeClient().DescribeClusterSecurity(request)
}

func (me *TkeService) CreateClusterAsGroup(ctx context.Context, id, groupPara, configPara string, labels []*tke.Label, iAdvanced InstanceAdvancedSettings) (asGroupId string, errRet error) {

	logId := getLogId(ctx)
	request := tke.NewCreateClusterAsGroupRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.AutoScalingGroupPara = &groupPara
	request.LaunchConfigurePara = &configPara
	if len(iAdvanced.ExtraArgs.Kubelet) > 0 {
		request.InstanceAdvancedSettings = &tke.InstanceAdvancedSettings{
			ExtraArgs: &iAdvanced.ExtraArgs,
		}
	}

	if len(labels) > 0 {
		request.Labels = labels
	}

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().CreateClusterAsGroup(request)
	if err != nil {
		errRet = err
		return
	}

	if response == nil || response.Response == nil || response.Response.AutoScalingGroupId == nil {
		errRet = fmt.Errorf("CreateClusterAsGroup return nil response")
		return
	}

	asGroupId = *response.Response.AutoScalingGroupId
	return
}

func (me *TkeService) DescribeClusterAsGroupsByGroupId(ctx context.Context, id string, groupId string) (clusterAsGroupSet *tke.ClusterAsGroup, errRet error) {
	logId := getLogId(ctx)
	request := tke.NewDescribeClusterAsGroupsRequest()

	request.ClusterId = &id
	request.AutoScalingGroupIds = []*string{&groupId}

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().DescribeClusterAsGroups(request)

	if err != nil {
		log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), err.Error())
		errRet = err
		return
	}

	if len(response.Response.ClusterAsGroupSet) > 0 {
		clusterAsGroupSet = response.Response.ClusterAsGroupSet[0]
	}
	return
}

func (me *TkeService) DeleteClusterAsGroups(ctx context.Context, id, asGroupId string) (errRet error) {

	logId := getLogId(ctx)
	request := tke.NewDeleteClusterAsGroupsRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.AutoScalingGroupIds = []*string{&asGroupId}

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseTkeClient().DeleteClusterAsGroups(request)
	if err != nil {
		errRet = err
	}
	return
}

/*
  for MANAGED_CLUSTER open internet access
*/
func (me *TkeService) CreateClusterEndpointVip(ctx context.Context, id string, securityPolicies []string) (errRet error) {
	logId := getLogId(ctx)

	request := tke.NewCreateClusterEndpointVipRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	if len(securityPolicies) > 0 {
		request.SecurityPolicies = make([]*string, 0, len(securityPolicies))
		for _, v := range securityPolicies {
			request.SecurityPolicies = append(request.SecurityPolicies, helper.String(v))
		}
	}

	ratelimit.Check(request.GetAction())

	_, err := me.client.UseTkeClient().CreateClusterEndpointVip(request)
	if err != nil {
		errRet = err
	}
	return
}

func (me *TkeService) DescribeClusterEndpointVipStatus(ctx context.Context, id string) (status string, message string, errRet error) {
	logId := getLogId(ctx)

	request := tke.NewDescribeClusterEndpointVipStatusRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id

	ratelimit.Check(request.GetAction())

	response, err := me.client.UseTkeClient().DescribeClusterEndpointVipStatus(request)
	if err != nil {
		errRet = err
		return
	}
	if response.Response == nil || response.Response.Status == nil {
		errRet = fmt.Errorf("sdk DescribeClusterEndpointVipStatus return empty status")
		return
	}
	status = *response.Response.Status
	message = *response.Response.ErrorMsg
	return
}

/*
  for INDEPENDENT_CLUSTER open internet access
*/
func (me *TkeService) CreateClusterEndpoint(ctx context.Context, id string, subnetId string, internet bool) (errRet error) {
	logId := getLogId(ctx)

	request := tke.NewCreateClusterEndpointRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id

	request.IsExtranet = &internet

	if subnetId != "" {
		request.SubnetId = &subnetId
	}

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseTkeClient().CreateClusterEndpoint(request)
	if err != nil {
		errRet = err
	}
	return
}

func (me *TkeService) DescribeClusterEndpointStatus(ctx context.Context, id string) (status string, message string, errRet error) {
	logId := getLogId(ctx)

	request := tke.NewDescribeClusterEndpointStatusRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id

	ratelimit.Check(request.GetAction())

	response, err := me.client.UseTkeClient().DescribeClusterEndpointStatus(request)
	if err != nil {
		errRet = err
		return
	}
	if response.Response == nil || response.Response.Status == nil {
		errRet = fmt.Errorf("sdk DescribeClusterEndpointStatus return empty status")
		return
	}
	status = *response.Response.Status
	message = status
	return
}

func (me *TkeService) DeleteClusterEndpoint(ctx context.Context, id string, isInternet bool) (errRet error) {
	logId := getLogId(ctx)
	request := tke.NewDeleteClusterEndpointRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.IsExtranet = &isInternet

	ratelimit.Check(request.GetAction())

	_, err := me.client.UseTkeClient().DeleteClusterEndpoint(request)
	if err != nil {
		errRet = err
		return
	}
	return
}

func (me *TkeService) DeleteClusterEndpointVip(ctx context.Context, id string) (errRet error) {
	logId := getLogId(ctx)
	request := tke.NewDeleteClusterEndpointVipRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id

	ratelimit.Check(request.GetAction())

	_, err := me.client.UseTkeClient().DeleteClusterEndpointVip(request)
	if err != nil {
		errRet = err
		return
	}
	return
}

func (me *TkeService) ModifyClusterEndpointSP(ctx context.Context, id string, securityPolicies []string) (errRet error) {
	logId := getLogId(ctx)
	request := tke.NewModifyClusterEndpointSPRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.SecurityPolicies = helper.Strings(securityPolicies)

	ratelimit.Check(request.GetAction())

	_, err := me.client.UseTkeClient().ModifyClusterEndpointSP(request)
	if err != nil {
		errRet = err
		return
	}
	return
}

func (me *TkeService) ModifyClusterAttribute(ctx context.Context, id string, projectId int64, clusterName string, clusterDesc string) (errRet error) {
	logId := getLogId(ctx)
	request := tke.NewModifyClusterAttributeRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.ProjectId = &projectId
	request.ClusterName = &clusterName
	request.ClusterDesc = &clusterDesc

	ratelimit.Check(request.GetAction())

	_, err := me.client.UseTkeClient().ModifyClusterAttribute(request)
	if err != nil {
		errRet = err
		return
	}
	return
}

func (me *TkeService) ModifyClusterVersion(ctx context.Context, id string, clusterVersion string, extraArgs interface{}) (errRet error) {
	logId := getLogId(ctx)
	request := tke.NewUpdateClusterVersionRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()

	if extraArgs != nil && len(extraArgs.([]interface{})) > 0 {
		// the first elem is in use
		extraInterface := extraArgs.([]interface{})
		extraMap := extraInterface[0].(map[string]interface{})

		kas := make([]*string, 0)
		if kaArgs, exist := extraMap["kube_apiserver"]; exist {
			args := kaArgs.([]interface{})
			for index := range args {
				str := args[index].(string)
				kas = append(kas, &str)
			}
		}
		kcms := make([]*string, 0)
		if kcmArgs, exist := extraMap["kube_controller_manager"]; exist {
			args := kcmArgs.([]interface{})
			for index := range args {
				str := args[index].(string)
				kcms = append(kcms, &str)
			}
		}
		kss := make([]*string, 0)
		if ksArgs, exist := extraMap["kube_scheduler"]; exist {
			args := ksArgs.([]interface{})
			for index := range args {
				str := args[index].(string)
				kss = append(kss, &str)
			}
		}

		request.ExtraArgs = &tke.ClusterExtraArgs{
			KubeAPIServer:         kas,
			KubeControllerManager: kcms,
			KubeScheduler:         kss,
		}
	}

	request.ClusterId = &id
	request.DstVersion = &clusterVersion

	ratelimit.Check(request.GetAction())

	_, err := me.client.UseTkeClient().UpdateClusterVersion(request)
	if err != nil {
		errRet = err
		return
	}
	return
}

func (me *TkeService) CheckClusterVersion(ctx context.Context, id string, clusterVersion string) (isOk bool, errRet error) {
	logId := getLogId(ctx)
	request := tke.NewDescribeAvailableClusterVersionRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	ratelimit.Check(request.GetAction())

	resp, err := me.client.UseTkeClient().DescribeAvailableClusterVersion(request)
	if err != nil {
		errRet = err
		return
	}

	if resp == nil || resp.Response == nil || resp.Response.Versions == nil {
		return
	}
	versions := resp.Response.Versions
	for _, v := range versions {
		if *v == clusterVersion {
			isOk = true
			return
		}
	}

	return
}

func (me *TkeService) CheckInstancesUpgradeAble(ctx context.Context, id string, upgradeType string) (instanceIds []string, errRet error) {
	logId := getLogId(ctx)
	request := tke.NewCheckInstancesUpgradeAbleRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.UpgradeType = &upgradeType
	ratelimit.Check(request.GetAction())

	resp, err := me.client.UseTkeClient().CheckInstancesUpgradeAble(request)
	if err != nil {
		errRet = err
		return
	}

	if resp == nil || resp.Response == nil || resp.Response.UpgradeAbleInstances == nil {
		return
	}
	for _, inst := range resp.Response.UpgradeAbleInstances {
		instanceIds = append(instanceIds, *inst.InstanceId)
	}

	return
}

func (me *TkeService) UpgradeClusterInstances(ctx context.Context, id string, upgradeType string, instanceIds []string) (errRet error) {
	logId := getLogId(ctx)
	request := tke.NewUpgradeClusterInstancesRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	op := "create"
	request.Operation = &op
	request.ClusterId = &id
	request.UpgradeType = &upgradeType
	request.InstanceIds = helper.Strings(instanceIds)
	ratelimit.Check(request.GetAction())

	_, err := me.client.UseTkeClient().UpgradeClusterInstances(request)
	if err != nil {
		errRet = err
		return
	}

	return
}

func (me *TkeService) DescribeImages(ctx context.Context) (imageIds []string, errRet error) {
	logId := getLogId(ctx)
	request := tke.NewDescribeImagesRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().DescribeImages(request)
	if err != nil {
		errRet = err
		return
	}

	for _, image := range response.Response.ImageInstanceSet {
		imageIds = append(imageIds, *image.ImageId)
	}
	return
}

func GetTkeLabels(d *schema.ResourceData, k string) []*tke.Label {
	labels := make([]*tke.Label, 0)
	if raw, ok := d.GetOk(k); ok {
		for k, v := range raw.(map[string]interface{}) {
			labels = append(labels, &tke.Label{Name: helper.String(k), Value: helper.String(v.(string))})
		}
	}
	return labels
}

func GetTkeTaints(d *schema.ResourceData, k string) []*tke.Taint {
	taints := make([]*tke.Taint, 0)
	if raw, ok := d.GetOk(k); ok {
		for _, v := range raw.([]interface{}) {
			vv := v.(map[string]interface{})
			taints = append(taints, &tke.Taint{Key: helper.String(vv["key"].(string)), Value: helper.String(vv["value"].(string)), Effect: helper.String(vv["effect"].(string))})
		}
	}
	return taints
}

func (me *TkeService) ModifyClusterAsGroupAttribute(ctx context.Context, id, asGroupId string, maxSize, minSize int64) (errRet error) {

	logId := getLogId(ctx)
	request := tke.NewModifyClusterAsGroupAttributeRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.ClusterAsGroupAttribute = &tke.ClusterAsGroupAttribute{
		AutoScalingGroupId: &asGroupId,
		AutoScalingGroupRange: &tke.AutoScalingGroupRange{
			MaxSize: &maxSize,
			MinSize: &minSize,
		},
	}

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseTkeClient().ModifyClusterAsGroupAttribute(request)
	if err != nil {
		errRet = err
	}
	return
}

func (me *TkeService) CreateClusterNodePool(ctx context.Context, clusterId, name, groupPara, configPara string, enableAutoScale bool, nodeOs string, nodeOsType string, labels []*tke.Label, taints []*tke.Taint, iAdvanced tke.InstanceAdvancedSettings) (asGroupId string, errRet error) {
	logId := getLogId(ctx)
	request := tke.NewCreateClusterNodePoolRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &clusterId
	request.Name = &name
	request.AutoScalingGroupPara = &groupPara
	request.LaunchConfigurePara = &configPara
	request.InstanceAdvancedSettings = &iAdvanced
	request.EnableAutoscale = &enableAutoScale
	request.NodePoolOs = &nodeOs
	request.OsCustomizeType = &nodeOsType

	if len(labels) > 0 {
		request.Labels = labels
	}

	if len(taints) > 0 {
		request.Taints = taints
	}

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().CreateClusterNodePool(request)
	if err != nil {
		errRet = err
		return
	}

	if response == nil || response.Response == nil || response.Response.NodePoolId == nil {
		errRet = fmt.Errorf("CreateClusterNodePool return nil response")
		return
	}

	asGroupId = *response.Response.NodePoolId
	return
}

func (me *TkeService) ModifyClusterNodePool(ctx context.Context, clusterId, nodePoolId string, name string, enableAutoScale bool, minSize int64, maxSize int64, nodeOs string, nodeOsType string, labels []*tke.Label, taints []*tke.Taint) (errRet error) {
	logId := getLogId(ctx)
	request := tke.NewModifyClusterNodePoolRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &clusterId
	request.NodePoolId = &nodePoolId
	request.Taints = taints
	request.Labels = labels
	request.EnableAutoscale = &enableAutoScale
	request.MaxNodesNum = &maxSize
	request.MinNodesNum = &minSize
	request.Name = &name
	request.OsName = &nodeOs
	request.OsCustomizeType = &nodeOsType

	if len(labels) > 0 {
		request.Labels = labels
	}

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseTkeClient().ModifyClusterNodePool(request)
	if err != nil {
		errRet = err
		return
	}
	return
}

func (me *TkeService) ModifyClusterNodePoolDesiredCapacity(ctx context.Context, clusterId, nodePoolId string, desiredCapacity int64) (errRet error) {
	logId := getLogId(ctx)
	request := tke.NewModifyNodePoolDesiredCapacityAboutAsgRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &clusterId
	request.NodePoolId = &nodePoolId
	request.DesiredCapacity = &desiredCapacity

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseTkeClient().ModifyNodePoolDesiredCapacityAboutAsg(request)
	if err != nil {
		errRet = err
		return
	}
	return
}

func (me *TkeService) DeleteClusterNodePool(ctx context.Context, id, nodePoolId string, deleteKeepInstance bool) (errRet error) {

	logId := getLogId(ctx)
	request := tke.NewDeleteClusterNodePoolRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()
	request.ClusterId = &id
	request.NodePoolIds = []*string{&nodePoolId}
	request.KeepInstance = &deleteKeepInstance

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseTkeClient().DeleteClusterNodePool(request)
	if err != nil {
		errRet = err
	}
	return
}

func (me *TkeService) DescribeNodePool(ctx context.Context, clusterId string, nodePoolId string) (
	nodePool *tke.NodePool,
	has bool,
	errRet error,
) {

	logId := getLogId(ctx)
	//the error code of cluster not exist is InternalError
	//check cluster exist first
	_, clusterHas, err := me.DescribeCluster(ctx, clusterId)
	if err != nil {
		errRet = err
		return
	}
	if !clusterHas {
		return
	}

	request := tke.NewDescribeClusterNodePoolDetailRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	request.ClusterId = helper.String(clusterId)
	request.NodePoolId = helper.String(nodePoolId)

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().DescribeClusterNodePoolDetail(request)

	if err != nil {
		errRet = err
		return
	}

	if response.Response.NodePool == nil {
		return
	}

	has = true
	nodePool = response.Response.NodePool

	return
}

//node pool global config
func (me *TkeService) ModifyClusterNodePoolGlobalConfig(ctx context.Context, request *tke.ModifyClusterAsGroupOptionAttributeRequest) (errRet error) {
	logId := getLogId(ctx)
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]\n", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseTkeClient().ModifyClusterAsGroupOptionAttribute(request)
	if err != nil {
		errRet = err
	}
	return
}

func (me *TkeService) DescribeClusterNodePoolGlobalConfig(ctx context.Context, clusterId string) (
	npGlobalConfig *tke.ClusterAsGroupOption,
	errRet error,
) {

	logId := getLogId(ctx)
	//the error code of cluster not exist is InternalError
	//check cluster exist first
	_, clusterHas, err := me.DescribeCluster(ctx, clusterId)
	if err != nil {
		errRet = err
		return
	}
	if !clusterHas {
		return
	}

	request := tke.NewDescribeClusterAsGroupOptionRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	request.ClusterId = helper.String(clusterId)

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseTkeClient().DescribeClusterAsGroupOption(request)

	if err != nil {
		errRet = err
		return
	}

	if response == nil || response.Response == nil || response.Response.ClusterAsGroupOption == nil {
		return
	}

	npGlobalConfig = response.Response.ClusterAsGroupOption

	return
}
