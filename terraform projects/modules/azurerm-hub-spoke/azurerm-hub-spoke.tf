terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
      version = ">=3.99.0"
    }
  }
}

provider "azurerm" {
  features {
  }
  skip_provider_registration = true
}

locals {

  ############################################
  ###### SIMPLE VARIABLES TRANSFORMATION #####
  ############################################

  tp_object = var.typology_object
  location = local.tp_object.location != null ? local.tp_object.location : "westeurope"
  vnet_cidr_notation_total = "/16"
  vnet_cidr_notation = "/24"
  vnet_cidr_block = ["10.0.0.0${local.vnet_cidr_notation_total}"]
  subnets_cidr_notation = local.tp_object.subnets_cidr_notation != null ? local.tp_object.subnets_cidr_notation : "/26"
  vpn_gateway_sku = "VpnGw2"
  #multiplicator = local.tp_object.multiplicator != null ? local.tp_object.multiplicator : 1
  rg_count = 1 + length(local.tp_object.spoke_objects) #* local.multiplicator
  env_name = local.tp_object.env_name != null ? local.tp_object.env_name : ""
  customer_name = local.tp_object.customer_name != null ? local.tp_object.customer_name : ""
  name_fix_pre = local.tp_object.name_prefix != null ? true : false
  name_fix = local.name_fix_pre ? local.name_fix_pre : local.tp_object.name_suffix != null ? false : false
  base_name = local.name_fix == null ? null : local.name_fix && local.tp_object.env_name != null ? "${local.tp_object.name_prefix}-${local.customer_name}-open-${local.env_name}" : local.name_fix == false && local.tp_object.env_name != null ? "${local.env_name}-${local.customer_name}-open-${local.tp_object.name_suffix}" : local.name_fix && local.tp_object.env_name == null ? "${local.tp_object.name_prefix}-${local.customer_name}-open" : local.name_fix == false && local.tp_object.env_name == null && local.tp_object.name_suffix != null ? "${local.customer_name}-open-${local.tp_object.name_suffix}" : null
  rg_name = local.name_fix ? "rg-${replace(local.base_name, "-open", "hub")}" : local.base_name != null ? "${replace(local.base_name, "-open", "hub")}-rg" : "rg-hub"
  vnet_base_name = local.name_fix ? "vnet-${replace(local.base_name, "-open", "hub")}" : local.base_name != null ? "${replace(local.base_name, "-open", "hub")}-vnet" : "vnet-hub"
  gateway_base_name = local.name_fix ? "gw-${replace(local.base_name, "-open", "hub")}" : local.base_name != null ? "${replace(local.base_name, "-open", "hub")}-gw" : "gw-hub-p2s"
  
  subnet_list_of_delegations = (jsondecode("{\"value\":[{\"name\":\"Microsoft.Web.serverFarms\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Web.serverFarms\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Web/serverFarms\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.ContainerInstance.containerGroups\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.ContainerInstance.containerGroups\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.ContainerInstance/containerGroups\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.Netapp.volumes\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Netapp.volumes\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Netapp/volumes\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.HardwareSecurityModules.dedicatedHSMs\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.HardwareSecurityModules.dedicatedHSMs\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.HardwareSecurityModules/dedicatedHSMs\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.ServiceFabricMesh.networks\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.ServiceFabricMesh.networks\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.ServiceFabricMesh/networks\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.Logic.integrationServiceEnvironments\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Logic.integrationServiceEnvironments\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Logic/integrationServiceEnvironments\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.Batch.batchAccounts\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Batch.batchAccounts\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Batch/batchAccounts\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.Sql.managedInstances\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Sql.managedInstances\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Sql/managedInstances\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\",\"Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action\",\"Microsoft.Network/virtualNetworks/subnets/unprepareNetworkPolicies/action\"]},{\"name\":\"Microsoft.Web.hostingEnvironments\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Web.hostingEnvironments\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Web/hostingEnvironments\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.BareMetal.CrayServers\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.BareMetal.CrayServers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.BareMetal/CrayServers\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Databricks.workspaces\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Databricks.workspaces\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Databricks/workspaces\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\",\"Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action\",\"Microsoft.Network/virtualNetworks/subnets/unprepareNetworkPolicies/action\"]},{\"name\":\"Microsoft.BareMetal.AzureHostedService\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.BareMetal.AzureHostedService\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.BareMetal/AzureHostedService\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.BareMetal.AzureVMware\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.BareMetal.AzureVMware\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.BareMetal/AzureVMware\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.StreamAnalytics.streamingJobs\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.StreamAnalytics.streamingJobs\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.StreamAnalytics/streamingJobs\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforPostgreSQL.serversv2\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.DBforPostgreSQL.serversv2\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforPostgreSQL/serversv2\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.AzureCosmosDB.clusters\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.AzureCosmosDB.clusters\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.AzureCosmosDB/clusters\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.MachineLearningServices.workspaces\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.MachineLearningServices.workspaces\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.MachineLearningServices/workspaces\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforPostgreSQL.singleServers\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.DBforPostgreSQL.singleServers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforPostgreSQL/singleServers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforPostgreSQL.flexibleServers\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.DBforPostgreSQL.flexibleServers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforPostgreSQL/flexibleServers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforMySQL.serversv2\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.DBforMySQL.serversv2\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforMySQL/serversv2\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforMySQL.flexibleServers\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.DBforMySQL.flexibleServers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforMySQL/flexibleServers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforMySQL.servers\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.DBforMySQL.servers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforMySQL/servers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.ApiManagement.service\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.ApiManagement.service\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.ApiManagement/service\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\",\"Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action\"]},{\"name\":\"Microsoft.Synapse.workspaces\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Synapse.workspaces\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Synapse/workspaces\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.PowerPlatform.vnetaccesslinks\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.PowerPlatform.vnetaccesslinks\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.PowerPlatform/vnetaccesslinks\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Network.dnsResolvers\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Network.dnsResolvers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Network/dnsResolvers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Kusto.clusters\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Kusto.clusters\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Kusto/clusters\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\",\"Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action\",\"Microsoft.Network/virtualNetworks/subnets/unprepareNetworkPolicies/action\"]},{\"name\":\"Microsoft.DelegatedNetwork.controller\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.DelegatedNetwork.controller\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DelegatedNetwork/controller\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.ContainerService.managedClusters\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.ContainerService.managedClusters\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.ContainerService/managedClusters\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.PowerPlatform.enterprisePolicies\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.PowerPlatform.enterprisePolicies\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.PowerPlatform/enterprisePolicies\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.StoragePool.diskPools\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.StoragePool.diskPools\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.StoragePool/diskPools\",\"actions\":[\"Microsoft.Network/virtualNetworks/read\"]},{\"name\":\"Microsoft.DocumentDB.cassandraClusters\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.DocumentDB.cassandraClusters\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DocumentDB/cassandraClusters\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Apollo.npu\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Apollo.npu\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Apollo/npu\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.AVS.PrivateClouds\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.AVS.PrivateClouds\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.AVS/PrivateClouds\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\"]},{\"name\":\"Microsoft.Orbital.orbitalGateways\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Orbital.orbitalGateways\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Orbital/orbitalGateways\",\"actions\":[\"Microsoft.Network/publicIPAddresses/join/action\",\"Microsoft.Network/virtualNetworks/subnets/join/action\",\"Microsoft.Network/virtualNetworks/read\",\"Microsoft.Network/publicIPAddresses/read\"]},{\"name\":\"Microsoft.Singularity.accounts.networks\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Singularity.accounts.networks\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Singularity/accounts/networks\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Singularity.accounts.npu\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Singularity.accounts.npu\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Singularity/accounts/npu\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.LabServices.labplans\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.LabServices.labplans\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.LabServices/labplans\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Fidalgo.networkSettings\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Fidalgo.networkSettings\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Fidalgo/networkSettings\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DevCenter.networkConnection\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.DevCenter.networkConnection\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DevCenter/networkConnection\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"NGINX.NGINXPLUS.nginxDeployments\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/NGINX.NGINXPLUS.nginxDeployments\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"NGINX.NGINXPLUS/nginxDeployments\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DevOpsInfrastructure.pools\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.DevOpsInfrastructure.pools\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DevOpsInfrastructure/pools\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.CloudTest.pools\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.CloudTest.pools\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.CloudTest/pools\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.CloudTest.hostedpools\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.CloudTest.hostedpools\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.CloudTest/hostedpools\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.CloudTest.images\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.CloudTest.images\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.CloudTest/images\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"PaloAltoNetworks.Cloudngfw.firewalls\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/PaloAltoNetworks.Cloudngfw.firewalls\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"PaloAltoNetworks.Cloudngfw/firewalls\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Qumulo.Storage.fileSystems\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Qumulo.Storage.fileSystems\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Qumulo.Storage/fileSystems\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.App.testClients\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.App.testClients\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.App/testClients\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.App.environments\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.App.environments\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.App/environments\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.ServiceNetworking.trafficControllers\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.ServiceNetworking.trafficControllers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.ServiceNetworking/trafficControllers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"GitHub.Network.networkSettings\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/GitHub.Network.networkSettings\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"GitHub.Network/networkSettings\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Network.networkWatchers\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Network.networkWatchers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Network/networkWatchers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Dell.Storage.fileSystems\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Dell.Storage.fileSystems\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Dell.Storage/fileSystems\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Netapp.scaleVolumes\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.Netapp.scaleVolumes\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Netapp/scaleVolumes\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Oracle.Database.networkAttachments\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Oracle.Database.networkAttachments\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Oracle.Database/networkAttachments\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"PureStorage.Block.storagePools\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/PureStorage.Block.storagePools\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"PureStorage.Block/storagePools\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Informatica.DataManagement.organizations\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Informatica.DataManagement.organizations\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Informatica.DataManagement/organizations\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.AzureCommunicationsGateway.networkSettings\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.AzureCommunicationsGateway.networkSettings\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.AzureCommunicationsGateway/networkSettings\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.PowerAutomate.hostedRpa\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.PowerAutomate.hostedRpa\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.PowerAutomate/hostedRpa\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.MachineLearningServices.workspaceComputes\",\"id\":\"/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/providers/Microsoft.Network/availableDelegations/Microsoft.MachineLearningServices.workspaceComputes\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.MachineLearningServices/workspaceComputes\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]}]}")).value

  ############################################
  ###### VARIABLE OBJECTS TRANSFORMATION #####
  ############################################

  rg_objects = {for each in [for a, b in range(local.rg_count) : {
    name = replace((a == local.rg_count - 1 && local.tp_object.hub_object.rg_name != null ? local.tp_object.hub_object.rg_name : local.rg_name != null && a == (local.rg_count - 1) ? local.rg_name : local.tp_object.spoke_objects[a].rg_name != null ? local.tp_object.spoke_objects[a].rg_name : replace(local.rg_name, "hub", "spoke${a + 1}")), "^-.+|.+-$", "/")
    location = local.tp_object.location != null ? local.tp_object.location : a == local.rg_count - 1 && local.tp_object.hub_object.location != null ? local.tp_object.hub_object.location : a != local.rg_count - 1 && local.tp_object.spoke_objects[a].location != null ? local.tp_object.spoke_objects[a].location : local.location
    solution_name = a == local.rg_count -1 ? null : can(local.tp_object.spoke_objects[a].solution_name) ? local.tp_object.spoke_objects[a].solution_name : null
    tags = a == local.rg_count - 1 && local.tp_object.hub_object.tags != null ? local.tp_object.hub_object.tags : a != local.rg_count - 1 ? local.tp_object.spoke_objects[a].tags : null
    vnet_name = local.vnet_objects_pre[a].name
  }] : each.name => each}

  vnet_objects_pre = [for a, b in range(local.rg_count) : {
    name = a == local.rg_count -1 && local.tp_object.hub_object.network == null ? local.vnet_base_name : a == local.rg_count -1 && local.tp_object.hub_object.network.vnet_name != null ? local.vnet_base_name : a == local.rg_count -1 && local.tp_object.hub_object.network.vnet_name == null ? local.vnet_base_name : a != local.rg_count - 1 && local.tp_object.spoke_objects[a].network == null ? replace(local.vnet_base_name, "hub", "spoke${a + 1}") : a != local.rg_count - 1 && local.tp_object.spoke_objects[a].network.vnet_name != null ? local.tp_object.spoke_objects[a].network.vnet_name : replace(local.vnet_base_name, "hub", "spoke${a + 1}")
    is_hub = a == local.rg_count - 1 ? true : false
    spoke_number = a != local.rg_count -1 ? a : null
    address_spaces = local.tp_object.address_spaces != null ? local.tp_object.address_spaces : a == local.rg_count -1 && local.tp_object.hub_object.network == null ? local.vnet_cidr_block : a == local.rg_count -1 && local.tp_object.hub_object.network.address_spaces != null ? local.tp_object.hub_object.network.address_spaces : a == local.rg_count -1 ? [cidrsubnet(local.vnet_cidr_block[0], 32 - tonumber(replace(local.vnet_cidr_notation, "/", "")), 0)] : a != local.rg_count -1 && !can(local.tp_object.spoke_objects[a].network.address_spaces) ? [cidrsubnet(local.vnet_cidr_block[0], 32 - tonumber(replace(local.vnet_cidr_notation, "/", "")), a + 1)] : a == local.rg_count -1 ? null : local.tp_object.spoke_objects[a].network.address_spaces != null ? local.tp_object.spoke_objects[a].network.address_spaces : [cidrsubnet(local.vnet_cidr_block[0], 32 - tonumber(replace(local.vnet_cidr_notation, "/", "")), a + 1)]
    solution_name = a == local.rg_count -1 ? null : can(local.tp_object.spoke_objects[a].solution_name) ? local.tp_object.spoke_objects[a].solution_name : null
    dns_servers = local.tp_object.dns_servers != null ? local.tp_object.dns_servers : a == local.rg_count - 1 && can(local.tp_object.hub_object.network.dns_servers) ? local.tp_object.hub_object.dns_servers : a != local.rg_count - 1 && can(local.tp_object.spoke_objects[a].network.dns_servers) ? local.tp_object.spoke_objects[a].network.dns_servers : null
    tags = local.tp_object.tags != null && can(local.tp_object.hub_object.network.tags) && a == local.rg_count -1 ? merge(local.tp_object.tags, local.tp_object.hub_object.network.tags) : local.tp_object.tags != null && a != local.rg_count -1 && can(local.tp_object.spoke_objects[a].network.tags) ? merge(local.tp_object.tags, local.tp_object.spoke_objects[a].network.tags) : local.tp_object.tags
    subnets = a == local.rg_count -1 && local.tp_object.hub_object.network == null ? null : a == local.rg_count -1 && can(local.tp_object.hub_object.network.subnet_objects) ? local.tp_object.hub_object.network.subnet_objects : a != local.rg_count -1 && local.tp_object.spoke_objects[a].network == null ? null : a != local.rg_count -1 && can(local.tp_object.spoke_objects[a].network.subnet_objects) ? local.tp_object.spoke_objects[a].network.subnet_objects : []
    ddos_protection_plan = can(local.tp_object.spoke_objects[a].network.ddos_protection_plan) ? local.tp_object.spoke_objects[a].network.ddos_protection_plan : null
  }]

  subnet_objects_pre = [for a, b in local.vnet_objects_pre : {
    subnets = b.subnets != null ? [for c, d in b.subnets : {
      name = d.name != null ? d.name : d.use_first_subnet != null && d.use_last_subnet == null ? replace(b.name, "vnet", "subnet${c + 1}") : replace(b.name, "vnet", "subnet${length(local.vnet_objects_pre[a].subnets) - c}")
      vnet_name = b.name
      solution_name = a == local.rg_count -1 ? null : can(local.tp_object.spoke_objects[a].solution_name) ? local.tp_object.spoke_objects[a].solution_name : null
      address_prefixes = d.address_prefixes != null ? d.address_prefixes : d.use_first_subnet != null && d.use_last_subnet == null ? [cidrsubnet(b.address_spaces[0], tonumber(replace(local.subnets_cidr_notation, "/", "")) - tonumber(replace(local.vnet_cidr_notation, "/", "")), c)] : [cidrsubnet(b.address_spaces[0], tonumber(replace(local.subnets_cidr_notation, "/", "")) - tonumber(replace(local.vnet_cidr_notation, "/", "")), pow((32 - tonumber(replace(local.subnets_cidr_notation, "/", "")) - (32 - tonumber(replace(local.vnet_cidr_notation, "/", "")))), 2) -1 -c)] 
      delegation = !can(d.delegation[0]) ? [] : [for f, g in range(length([for h, i in local.subnet_list_of_delegations : i.serviceName if can(regexall(lower(d.delegation[0].service_name_pattern), lower(i.serviceName))[0])])) : {
        name = split(".", [for h, i in local.subnet_list_of_delegations : i.serviceName if can(regexall(lower(d.delegation[0].service_name_pattern), lower(i.serviceName))[0])][f])[1]
        service_name = [for h, i in local.subnet_list_of_delegations : i.serviceName if can(regexall(lower(d.delegation[0].service_name_pattern), lower(i.serviceName))[0])][f]
        actions = [for h, i in local.subnet_list_of_delegations : i.actions if can(regexall(lower(d.delegation[0].service_name_pattern), lower(i.serviceName))[0])][f]
      }] 
    }] : null
  }]

  peering_objects_from_hub_to_spokes = [for a, b in range(length(local.vnet_objects_pre) -1) : {
    name = local.tp_object.hub_object.network == null ? "peering-from-hub-to-spoke${a + 1}" : local.tp_object.hub_object.network.vnet_peering_name != null ? "${local.tp_object.hub_object.network.vnet_peering_name}${a}" : "peering-from-hub-to-spoke${a + 1}"
    vnet_name = [for c, d in local.vnet_objects_pre : d.name if d.is_hub][0]
    remote_virtual_network_id = [for c, d in local.vnet_return_helper_objects : d.id if d.address_space[0] == local.vnet_objects_pre[a].address_spaces[0]][0]
    allow_virtual_network_access = local.tp_object.hub_object.network == null ? true : local.tp_object.hub_object.network.vnet_peering_allow_virtual_network_access != null ? local.tp_object.hub_object.network.vnet_peering_allow_virtual_network_access : false
    allow_forwarded_traffic = local.tp_object.hub_object.network == null ? false : local.tp_object.hub_object.network.vnet_peering_allow_forwarded_traffic != null ? local.tp_object.hub_object.network.vnet_peering_allow_forwarded_traffic : true
    allow_gateway_transit = true
    use_remote_gateways = false
    solution_name = null
  }]

  peering_objects_from_spokes_to_hub = [for a, b in range(length(local.vnet_objects_pre) -1) : {
    name = local.tp_object.spoke_objects[a].network == null ? "peering-from-spoke${a + 1}-to-hub" : local.tp_object.spoke_objects[a].network.vnet_peering_name != null ? "${local.tp_object.spoke_objects[a].network.vnet_peering_name}${a}" : "peering-from-spoke${a + 1}-to-hub"
    vnet_name = local.vnet_objects_pre[a].name
    remote_virtual_network_id = [for c, d in local.vnet_return_helper_objects : d.id if d.address_space[0] == ([for e, f in local.vnet_objects_pre : f.address_spaces[0] if f.is_hub])[0]][0]
    allow_virtual_network_access = local.tp_object.spoke_objects[a].network == null ? false : local.tp_object.spoke_objects[a].network.vnet_peering_allow_virtual_network_access != null ? local.tp_object.spoke_objects[a].network.vnet_peering_allow_virtual_network_access : false
    allow_forwarded_traffic = local.tp_object.spoke_objects[a].network == null ? true : local.tp_object.spoke_objects[a].network.vnet_peering_allow_forwarded_traffic != null ? local.tp_object.spoke_objects[a].network.vnet_peering_allow_forwarded_traffic : true
    allow_gateway_transit = false
    use_remote_gateways = false
    solution_name = null
  }]

  gateway_object = local.tp_object.hub_object.network == null ? {} : local.tp_object.hub_object.network.vpn != null || local.tp_object.hub_object.network.vpn == {} ? {
    name = local.tp_object.hub_object.network.vpn.gw_name == null ? local.gateway_base_name : local.tp_object.hub_object.network.vpn.gw_name
    sku = local.tp_object.hub_object.network.vpn.gw_sku == null ? local.vpn_gateway_sku : local.tp_object.hub_object.network.vpn.gw_sku
    type = "Vpn"
    remote_vnet_traffic_enabled = true

    ip_configuration = {
      subnet_id = [for a, b in local.subnet_return_helper_objects : a.id if a.name == "GatewaySubnet"][0]
    }
    
  } : {}

  pip_gw_object = local.gateway_object != {} ? {
    name = local.tp_object.hub_object.network.vpn.
  }

  vnet_objects = {for each in local.vnet_objects_pre : each.name => each}
  subnet_objects = {for each in flatten([for a in local.subnet_objects_pre : a.subnets if a.subnets != null]) : each.name => each}
  peering_objects = {for each in flatten([local.peering_objects_from_hub_to_spokes, local.peering_objects_from_spokes_to_hub]) : each.name => each }

  ############################################
  ########## VARIABLE RETURN OBJECTS #########
  ############################################

  rg_return_helper_objects = local.rg_return_object != {} ? values(local.rg_return_object) : []
  rg_return_object = azurerm_resource_group.rg_object
  vnet_return_objects = azurerm_virtual_network.vnet_object
  vnet_return_helper_objects = local.vnet_return_objects != {} ? values(local.vnet_return_objects) : []
  subnet_return_objects = azurerm_subnet.subnet_object
  subnet_return_helper_objects = azurerm_subnet.subnet_object != {} ? values(local.subnet_return_objects) : []
  peering_return_objects = azurerm_virtual_network_peering.peering_object
}

  ############################################
  ############ RESOURCE DEFINITIONS ##########
  ############################################

resource "azurerm_resource_group" "rg_object" {
  for_each = local.rg_objects
  name = each.value.solution_name == null ? each.key : replace(each.key, "spoke", "${each.value.solution_name}-spoke")
  location = each.value.location
}

resource "azurerm_virtual_network" "vnet_object" {
  for_each = local.vnet_objects
  name = each.value.solution_name == null ? each.key : replace(each.key, "spoke", "${each.value.solution_name}-spoke")
  location = [for a in local.rg_objects : a.location if a.vnet_name == each.key][0]
  resource_group_name = each.value.solution_name == null ? [for a in local.rg_objects : a.name if a.vnet_name == each.key][0] : replace(replace(each.key, "spoke", "${each.value.solution_name}-spoke"), "vnet", "rg")
  address_space = each.value.address_spaces
  dns_servers = each.value.dns_servers
  tags = each.value.tags
  
  dynamic "ddos_protection_plan" {
    for_each = each.value.ddos_protection_plan != null ? {for a in [each.value.ddos_protection_plan] : a.id => a} : {}
    content {
      id = ddos_protection_plan.key
      enable = ddos_protection_plan.value.enable
    }
  }

  depends_on = [ azurerm_resource_group.rg_object ]
}

resource "azurerm_subnet" "subnet_object" {
  for_each = local.subnet_objects
  name = each.key
  resource_group_name = each.value.solution_name == null ? [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0] : replace(replace(each.value.vnet_name, "spoke", "${each.value.solution_name}-spoke"), "vnet", "rg")
  virtual_network_name = each.value.vnet_name
  address_prefixes = each.value.address_prefixes
  service_endpoints = can(each.value.service_endpoints) ? each.value.service_endpoints : null
  service_endpoint_policy_ids = can(each.value.service_endpoint_policy_ids) ? each.value.service_endpoint_policy_ids : null

  dynamic "delegation" {
    for_each = each.value.delegation
    content {
      name = delegation.value.name

      service_delegation {
        name = delegation.value.service_name
        actions = delegation.value.actions
      }
    }
  }
  
  depends_on = [ azurerm_virtual_network.vnet_object ]
}

resource "azurerm_virtual_network_peering" "peering_object" {
  for_each = local.peering_objects
  name = each.key
  virtual_network_name = each.value.vnet_name
  remote_virtual_network_id = each.value.remote_virtual_network_id
  resource_group_name = each.value.solution_name == null ? [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0] : replace(replace(each.value.vnet_name, "spoke", "${each.value.solution_name}-spoke"), "vnet", "rg")
  allow_virtual_network_access = each.value.allow_virtual_network_access
  allow_forwarded_traffic = each.value.allow_forwarded_traffic
  allow_gateway_transit = each.value.allow_gateway_transit
  use_remote_gateways = each.value.use_remote_gateways
}

resource "azurerm_public_ip" "gw_pip_object" {
  name = 
}

output "vnet" {
  value = local.vnet_objects_pre
}