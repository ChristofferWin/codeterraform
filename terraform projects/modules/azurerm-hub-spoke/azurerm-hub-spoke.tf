terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
      version = ">=3.99.0"
      configuration_aliases = [ azurerm.hub, azurerm.spoke ]
    }
  }
}

################################################################################################
######################################### NOTES ################################################
##                                                                                            ##
##  Date: 17-07-2024                                                                          ## 
##  State: Version 2.0.0                                                                      ##
##  Missing: General clean up of local variable definitions                                   ##
##  Improvements (1): Making it possible to create hub / spokes in different Azure contexts   ##                                                                                
##  =||= (2): Making comments / MARK:´s in the file for clean up                              ## 
##  =||= (3): N/A                                                                             ##
##  Future improvements: N/A                                                                  ##
##                                                                                            ##
## -------------------------------------------------------------------------------------------##
##                                                                                            ##
##  Date: 04-06-2024                                                                          ## 
##  State: Version 1.0.0                                                                      ##
##  Missing: As part of version 1.1, more support for the use of custom names will be added   ##
##  Improvements (1): N/A                                                                     ##                                                                                
##  =||= (2): N/A                                                                             ## 
##  =||= (3): N/A                                                                             ##
##  Future improvements: N/A                                                                  ##
##                                                                                            ##
################################################################################################


locals {

  ############################################
  ###### SIMPLE VARIABLES TRANSFORMATION #####
  ############################################ 

  //MARK: DEFINE DEFAULT VALUES
  tp_object = var.topology_object
  tenant_id = can(data.azurerm_client_config.context_object[0].tenant_id) ? data.azurerm_client_config.context_object[0].tenant_id : null
  vnet_cidr_notation = can(local.tp_object.hub_object.network.address_spaces[0]) ? "/${split("/", local.tp_object.hub_object.network.address_spaces[0])[1]}" : "/24"
  vnet_cidr_notation_number_difference = tonumber(replace(local.vnet_cidr_notation, "/", "")) < 24 ? 32 - tonumber(replace(local.vnet_cidr_notation, "/", "")) - 8 : tonumber(replace(local.vnet_cidr_notation, "/", "")) > 24 ? (32 - tonumber(replace(local.vnet_cidr_notation, "/", "")) - 8) * -1 : 0
  vnet_cidr_total = ["10.0.0.0/16"]
  subnets_cidr_notation = local.tp_object.subnets_cidr_notation != null ? local.tp_object.subnets_cidr_notation : "/26"
  vpn_gateway_sku = "VpnGw2"
  create_firewall = local.tp_object.hub_object.network == null ? false : local.tp_object.hub_object.network.firewall != null ? true : false
  create_vpn = local.tp_object.hub_object.network == null ? false : local.tp_object.hub_object.network.vpn != null ? true : false
  rg_count = !can(length(local.tp_object.spoke_objects)) ? 1 : can(length(local.tp_object.spoke_objects)) ? length(local.tp_object.spoke_objects) + 1 : 1
  env_name = local.tp_object.env_name != null ? local.tp_object.env_name : ""
  project_name = local.tp_object.project_name != null ? local.tp_object.project_name : "" 
  name_fix_pre = local.tp_object.name_prefix != null ? true : false
  name_fix = local.name_fix_pre ? local.name_fix_pre : local.tp_object.name_suffix != null ? false : false
  base_name = local.name_fix && local.tp_object.env_name != null ? "${local.tp_object.name_prefix}-${local.project_name}-open-${local.env_name}" : local.name_fix == false && local.tp_object.env_name != null ? "${local.env_name}-${local.project_name}-open-${local.tp_object.name_suffix}" : local.name_fix && local.tp_object.env_name == null ? "${local.tp_object.name_prefix}-open" : local.name_fix == false && local.tp_object.env_name == null && local.tp_object.name_suffix != null ? "open-${local.tp_object.name_suffix}" : null
  rg_name = local.name_fix ? "-rg-${replace(local.base_name, "-open", "-hub")}" : local.base_name != null ? "${replace(local.base_name, "open-", "hub-")}-rg" : "rg-hub"
  vnet_base_name = local.name_fix ? "vnet-${replace(local.base_name, "-open", "-hub")}" : local.base_name != null ? "${replace(local.base_name, "open-", "hub-")}-vnet" : "vnet-hub"
  gateway_base_name = local.name_fix ? "gw-${replace(local.base_name, "-open", "-hub")}" : local.base_name != null ? "${replace(local.base_name, "open-", "hub-")}-gw" : "gw-hub-p2s"
  pip_count = local.create_firewall && local.create_vpn ? 2 : local.create_firewall || local.create_vpn ? 1 : 0
  subnet_list_of_delegations = (jsondecode("{\"value\":[{\"name\":\"Microsoft.Web.serverFarms\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Web.serverFarms\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Web/serverFarms\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.ContainerInstance.containerGroups\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.ContainerInstance.containerGroups\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.ContainerInstance/containerGroups\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.Netapp.volumes\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Netapp.volumes\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Netapp/volumes\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.HardwareSecurityModules.dedicatedHSMs\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.HardwareSecurityModules.dedicatedHSMs\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.HardwareSecurityModules/dedicatedHSMs\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.ServiceFabricMesh.networks\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.ServiceFabricMesh.networks\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.ServiceFabricMesh/networks\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.Logic.integrationServiceEnvironments\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Logic.integrationServiceEnvironments\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Logic/integrationServiceEnvironments\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.Batch.batchAccounts\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Batch.batchAccounts\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Batch/batchAccounts\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.Sql.managedInstances\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Sql.managedInstances\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Sql/managedInstances\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\",\"Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action\",\"Microsoft.Network/virtualNetworks/subnets/unprepareNetworkPolicies/action\"]},{\"name\":\"Microsoft.Web.hostingEnvironments\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Web.hostingEnvironments\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Web/hostingEnvironments\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/action\"]},{\"name\":\"Microsoft.BareMetal.CrayServers\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.BareMetal.CrayServers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.BareMetal/CrayServers\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Databricks.workspaces\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Databricks.workspaces\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Databricks/workspaces\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\",\"Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action\",\"Microsoft.Network/virtualNetworks/subnets/unprepareNetworkPolicies/action\"]},{\"name\":\"Microsoft.BareMetal.AzureHostedService\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.BareMetal.AzureHostedService\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.BareMetal/AzureHostedService\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.BareMetal.AzureVMware\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.BareMetal.AzureVMware\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.BareMetal/AzureVMware\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.StreamAnalytics.streamingJobs\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.StreamAnalytics.streamingJobs\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.StreamAnalytics/streamingJobs\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforPostgreSQL.serversv2\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.DBforPostgreSQL.serversv2\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforPostgreSQL/serversv2\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.AzureCosmosDB.clusters\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.AzureCosmosDB.clusters\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.AzureCosmosDB/clusters\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.MachineLearningServices.workspaces\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.MachineLearningServices.workspaces\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.MachineLearningServices/workspaces\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforPostgreSQL.singleServers\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.DBforPostgreSQL.singleServers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforPostgreSQL/singleServers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforPostgreSQL.flexibleServers\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.DBforPostgreSQL.flexibleServers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforPostgreSQL/flexibleServers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforMySQL.serversv2\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.DBforMySQL.serversv2\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforMySQL/serversv2\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforMySQL.flexibleServers\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.DBforMySQL.flexibleServers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforMySQL/flexibleServers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DBforMySQL.servers\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.DBforMySQL.servers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DBforMySQL/servers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.ApiManagement.service\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.ApiManagement.service\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.ApiManagement/service\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\",\"Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action\"]},{\"name\":\"Microsoft.Synapse.workspaces\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Synapse.workspaces\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Synapse/workspaces\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.PowerPlatform.vnetaccesslinks\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.PowerPlatform.vnetaccesslinks\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.PowerPlatform/vnetaccesslinks\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Network.dnsResolvers\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Network.dnsResolvers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Network/dnsResolvers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Kusto.clusters\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Kusto.clusters\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Kusto/clusters\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\",\"Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action\",\"Microsoft.Network/virtualNetworks/subnets/unprepareNetworkPolicies/action\"]},{\"name\":\"Microsoft.DelegatedNetwork.controller\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.DelegatedNetwork.controller\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DelegatedNetwork/controller\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.ContainerService.managedClusters\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.ContainerService.managedClusters\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.ContainerService/managedClusters\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.PowerPlatform.enterprisePolicies\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.PowerPlatform.enterprisePolicies\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.PowerPlatform/enterprisePolicies\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.StoragePool.diskPools\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.StoragePool.diskPools\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.StoragePool/diskPools\",\"actions\":[\"Microsoft.Network/virtualNetworks/read\"]},{\"name\":\"Microsoft.DocumentDB.cassandraClusters\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.DocumentDB.cassandraClusters\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DocumentDB/cassandraClusters\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Apollo.npu\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Apollo.npu\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Apollo/npu\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.AVS.PrivateClouds\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.AVS.PrivateClouds\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.AVS/PrivateClouds\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\"]},{\"name\":\"Microsoft.Orbital.orbitalGateways\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Orbital.orbitalGateways\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Orbital/orbitalGateways\",\"actions\":[\"Microsoft.Network/publicIPAddresses/join/action\",\"Microsoft.Network/virtualNetworks/subnets/join/action\",\"Microsoft.Network/virtualNetworks/read\",\"Microsoft.Network/publicIPAddresses/read\"]},{\"name\":\"Microsoft.Singularity.accounts.networks\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Singularity.accounts.networks\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Singularity/accounts/networks\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Singularity.accounts.npu\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Singularity.accounts.npu\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Singularity/accounts/npu\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.LabServices.labplans\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.LabServices.labplans\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.LabServices/labplans\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Fidalgo.networkSettings\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Fidalgo.networkSettings\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Fidalgo/networkSettings\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DevCenter.networkConnection\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.DevCenter.networkConnection\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DevCenter/networkConnection\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"NGINX.NGINXPLUS.nginxDeployments\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/NGINX.NGINXPLUS.nginxDeployments\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"NGINX.NGINXPLUS/nginxDeployments\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.DevOpsInfrastructure.pools\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.DevOpsInfrastructure.pools\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.DevOpsInfrastructure/pools\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.CloudTest.pools\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.CloudTest.pools\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.CloudTest/pools\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.CloudTest.hostedpools\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.CloudTest.hostedpools\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.CloudTest/hostedpools\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.CloudTest.images\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.CloudTest.images\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.CloudTest/images\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"PaloAltoNetworks.Cloudngfw.firewalls\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/PaloAltoNetworks.Cloudngfw.firewalls\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"PaloAltoNetworks.Cloudngfw/firewalls\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Qumulo.Storage.fileSystems\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Qumulo.Storage.fileSystems\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Qumulo.Storage/fileSystems\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.App.testClients\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.App.testClients\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.App/testClients\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.App.environments\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.App.environments\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.App/environments\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.ServiceNetworking.trafficControllers\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.ServiceNetworking.trafficControllers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.ServiceNetworking/trafficControllers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"GitHub.Network.networkSettings\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/GitHub.Network.networkSettings\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"GitHub.Network/networkSettings\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Network.networkWatchers\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Network.networkWatchers\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Network/networkWatchers\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Dell.Storage.fileSystems\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Dell.Storage.fileSystems\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Dell.Storage/fileSystems\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.Netapp.scaleVolumes\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.Netapp.scaleVolumes\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.Netapp/scaleVolumes\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Oracle.Database.networkAttachments\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Oracle.Database.networkAttachments\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Oracle.Database/networkAttachments\",\"actions\":[\"Microsoft.Network/networkinterfaces/*\",\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"PureStorage.Block.storagePools\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/PureStorage.Block.storagePools\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"PureStorage.Block/storagePools\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Informatica.DataManagement.organizations\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Informatica.DataManagement.organizations\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Informatica.DataManagement/organizations\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.AzureCommunicationsGateway.networkSettings\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.AzureCommunicationsGateway.networkSettings\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.AzureCommunicationsGateway/networkSettings\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.PowerAutomate.hostedRpa\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.PowerAutomate.hostedRpa\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.PowerAutomate/hostedRpa\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]},{\"name\":\"Microsoft.MachineLearningServices.workspaceComputes\",\"id\":\"/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Network/availableDelegations/Microsoft.MachineLearningServices.workspaceComputes\",\"type\":\"Microsoft.Network/availableDelegations\",\"serviceName\":\"Microsoft.MachineLearningServices/workspaceComputes\",\"actions\":[\"Microsoft.Network/virtualNetworks/subnets/join/action\"]}]}")).value

  ############################################
  ###### VARIABLE OBJECTS TRANSFORMATION #####
  ############################################

//MARK: OBJECT TRANSFORMATION
  rg_objects = {for each in [for a, b in range(local.rg_count) : {
    name = replace(replace(replace((a == local.rg_count - 1 && local.tp_object.hub_object.rg_name != null ? local.tp_object.hub_object.rg_name : local.rg_name != null && a == (local.rg_count - 1) ? local.rg_name : local.tp_object.spoke_objects[a].rg_name != null ? local.tp_object.spoke_objects[a].rg_name : replace(local.rg_name, "hub", "spoke${a + 1}")), "^-.+|.+-$", "/"), "/(^-|-$)/", ""), "--", "-")
    location = local.tp_object.location != null ? local.tp_object.location : a == local.rg_count - 1 && local.tp_object.hub_object.location != null ? local.tp_object.hub_object.location : !can(local.tp_object.spoke_objects[a].location) ? "westeurope" : local.tp_object.spoke_objects[a].location != null ? local.tp_object.spoke_objects[a].location : "westeurope"
    vnet_name = local.vnet_objects_pre[a].name
    tags = a == local.rg_count -1 ? merge(local.tp_object.tags, local.tp_object.hub_object.tags) : merge(local.tp_object.tags, local.tp_object.spoke_objects[a].tags)
  }] : each.name => each}

  vnet_objects_pre = [for a, b in range(local.rg_count) : {
    name = replace(replace(a == local.rg_count -1 && local.tp_object.hub_object.network == null ? local.vnet_base_name : a == local.rg_count -1 && local.tp_object.hub_object.network.vnet_name != null ? local.tp_object.hub_object.network.vnet_name : a == local.rg_count -1 && local.tp_object.hub_object.network.vnet_name == null ? local.vnet_base_name : a != local.rg_count - 1 && local.tp_object.spoke_objects[a].network == null ? replace(local.vnet_base_name, "hub", "spoke${a + 1}") : a != local.rg_count - 1 && local.tp_object.spoke_objects[a].network.vnet_name != null ? local.tp_object.spoke_objects[a].network.vnet_name : replace(local.vnet_base_name, "hub", "spoke${a + 1}"), "/(|^-|-$)/", ""), "--", "-")
    is_hub = a == local.rg_count - 1 ? true : false
    spoke_number = a != local.rg_count -1 ? a : null
    address_spaces = a == local.rg_count -1 && !can(local.tp_object.hub_object.network.address_spaces[0]) ? [cidrsubnet(local.vnet_cidr_total[0], 32 - tonumber(replace(local.vnet_cidr_notation, "/", "")), 0)] : a == local.rg_count -1 && local.tp_object.hub_object.network.address_spaces != null ? local.tp_object.hub_object.network.address_spaces : a == local.rg_count -1 ? [cidrsubnet(local.vnet_cidr_total[0], 32 - tonumber(replace(local.vnet_cidr_notation, "/", "")), 0)] : a != local.rg_count -1 && can(local.tp_object.spoke_objects[a].network.address_spaces[0]) ? [local.tp_object.spoke_objects[a].network.address_spaces[0]] : a != local.rg_count -1 ? [cidrsubnet(local.vnet_cidr_total[0], 32 - tonumber(replace(local.vnet_cidr_notation, "/", "")) - local.vnet_cidr_notation_number_difference, a + 1)] : null
    tags = a == local.rg_count -1 ? merge(local.tp_object.tags, local.tp_object.hub_object.network.tags) : merge(local.tp_object.tags, local.tp_object.spoke_objects[a].network.tags)
    dns_servers = local.tp_object.dns_servers != null ? local.tp_object.dns_servers : a == local.rg_count - 1 && can(local.tp_object.hub_object.network.dns_servers[0]) ? local.tp_object.hub_object.network.dns_servers : a != local.rg_count - 1 && can(local.tp_object.spoke_objects[a].network.dns_servers) ? local.tp_object.spoke_objects[a].network.dns_servers : null
    subnets = a == local.rg_count -1 && local.tp_object.hub_object.network == null ? null : a == local.rg_count -1 && can(local.tp_object.hub_object.network.subnet_objects) ? local.tp_object.hub_object.network.subnet_objects : a != local.rg_count -1 && local.tp_object.spoke_objects[a].network == null ? null : a != local.rg_count -1 && can(local.tp_object.spoke_objects[a].network.subnet_objects) ? local.tp_object.spoke_objects[a].network.subnet_objects : []
    ddos_protection_plan = can(local.tp_object.spoke_objects[a].network.ddos_protection_plan) ? local.tp_object.spoke_objects[a].network.ddos_protection_plan : null
  }]

  wan_object = !can(local.tp_object.hub_object.network.wan) ? {} : local.tp_object.hub_object.network.wan != null ? {for each in [for a , b in [local.tp_object.hub_object.network.wan] : {
    name = b.name != null ? b.name : replace(local.vnet_base_name, "vnet", "wan")
  }] : each.name => each} : {}

  subnet_objects_pre = [for a, b in local.vnet_objects_pre : {
    subnets = can(flatten(b.*.subnets)) ? [for c, d in ([for e, f in flatten(b.*.subnets) : f if f != null]) : {
      name = !can(d.name) ? replace(b.name, "vnet", "subnet${c + 1}") : d.name != null ? d.name : replace(b.name, "vnet", "subnet${c + 1}")
      name_unique = !can(d.name) ? replace(b.name, "vnet", "subnet${c + 1}-${c}-unique") : d.name != null ? "${d.name}-${c}-unique" : replace(b.name, "vnet", "subnet${c + 1}-${c}-unique")
      vnet_name = b.name
      address_prefix = can(d.address_prefix[0]) ? d.address_prefix : d.use_first_subnet != null && d.use_last_subnet == null && a == local.rg_count -1 ? [cidrsubnet(b.address_spaces[0], tonumber(replace(local.subnets_cidr_notation, "/", "")) - tonumber(split("/", b.address_spaces[0])[1]), c)] : d.use_first_subnet == null && d.use_last_subnet != null ? [cidrsubnet(b.address_spaces[0], tonumber(replace(local.subnets_cidr_notation, "/", "")) - tonumber(split("/", b.address_spaces[0])[1]), pow((32 - tonumber(replace(local.subnets_cidr_notation, "/", "")) - (32 - tonumber(split("/", b.address_spaces[0])[1]))), 2) -1 -c)] : [cidrsubnet(b.address_spaces[0], tonumber(replace(local.subnets_cidr_notation, "/", "")) - tonumber(split("/", b.address_spaces[0])[1]), c)]

      delegation = c == 0 && can(d.delegation[0]) ? [for f, g in range(length([for h, i in local.subnet_list_of_delegations : i.serviceName if can(regexall(lower(d.delegation[0].service_name_pattern), lower(i.serviceName))[0])])) : {
      name = split(".", [for h, i in local.subnet_list_of_delegations : i.serviceName if can(regexall(lower(d.delegation[0].service_name_pattern), lower(i.serviceName))[0])][f])[1]
      service_name = [for h, i in local.subnet_list_of_delegations : i.serviceName if can(regexall(lower(d.delegation[0].service_name_pattern), lower(i.serviceName))[0])][f]
      actions = [for h, i in local.subnet_list_of_delegations : i.actions if can(regexall(lower(d.delegation[0].service_name_pattern), lower(i.serviceName))[0])][f]
      }] : []
    }] : null
  }]

  peering_objects_from_hub_to_spokes = local.tp_object.hub_object.network.vnet_spoke_address_spaces != null || can(length(local.tp_object.spoke_objects)) ? [for a, b in range(length(local.vnet_objects_pre) -1) : {
    name = local.tp_object.hub_object.network.vnet_peering_name != null ? "${local.tp_object.hub_object.network.vnet_peering_name}${a + 1}" : "peering-from-hub-to-spoke-${split("-", data.azurerm_client_config.context_spoke_object[0].subscription_id)[0]}-${a + 1}"
    vnet_name = local.tp_object.hub_object.network.vnet_resource_id == null ? [for c, d in local.vnet_objects_pre : d.name if d.is_hub][0] : null
    tags = a == local.rg_count -1 ? merge(local.tp_object.tags, local.tp_object.hub_object.network.tags) : merge(local.tp_object.tags, local.tp_object.spoke_objects[a].network.tags)
    remote_virtual_network_id = [for c, d in local.vnet_return_helper_objects : d.id if d.address_space[0] == local.vnet_objects_pre[a].address_spaces[0]][0]
    allow_virtual_network_access = local.tp_object.hub_object.network == null ? true : local.tp_object.hub_object.network.vnet_peering_allow_virtual_network_access != null ? local.tp_object.hub_object.network.vnet_peering_allow_virtual_network_access : true
    allow_forwarded_traffic = local.tp_object.hub_object.network == null ? true : local.tp_object.hub_object.network.vnet_peering_allow_forwarded_traffic != null ? local.tp_object.hub_object.network.vnet_peering_allow_forwarded_traffic : true
    allow_gateway_transit = true
    use_remote_gateways = false
  }] : []

  peering_objects_from_spokes_to_hub = [for a, b in range(length(local.vnet_objects_pre) -1) : {
    name = local.tp_object.spoke_objects[a].network.vnet_peering_name != null ? "${local.tp_object.spoke_objects[a].network.vnet_peering_name}${a}" : "peering-from-spoke${a + 1}-to-hub"
    vnet_name = local.vnet_objects_pre[a].name
    tags = a == local.rg_count -1 ? merge(local.tp_object.tags, local.tp_object.hub_object.network.tags) : merge(local.tp_object.tags, local.tp_object.spoke_objects[a].network.tags)
    remote_virtual_network_id = local.tp_object.hub_object.network.vnet_resource_id != null ? local.tp_object.hub_object.network.vnet_resource_id : [for c, d in local.vnet_return_helper_objects : d.id if d.address_space[0] == ([for e, f in local.vnet_objects_pre : f.address_spaces[0] if f.is_hub])[0]][0]
    allow_virtual_network_access = local.tp_object.spoke_objects[a].network == null ? true : local.tp_object.spoke_objects[a].network.vnet_peering_allow_virtual_network_access != null ? local.tp_object.spoke_objects[a].network.vnet_peering_allow_virtual_network_access : true
    allow_forwarded_traffic = local.tp_object.spoke_objects[a].network == null ? true : local.tp_object.spoke_objects[a].network.vnet_peering_allow_forwarded_traffic != null ? local.tp_object.spoke_objects[a].network.vnet_peering_allow_forwarded_traffic : true
    allow_gateway_transit = false
    use_remote_gateways = local.gw_object != {} ? true : false
  }]
  
  route_table_objects_pre = (local.wan_object == {} && !can(local.tp_object.hub_object.network.firewall)) ? [] : local.tp_object.hub_object.network.firewall != null || local.tp_object.hub_object.network.fw_resource_id != null || local.tp_object.hub_object.network.fw_private_ip != null ? [for a, b in flatten([for c, d in values(local.subnet_objects) : d if d.vnet_name != [for e, f in local.vnet_objects_pre : f.name if e == local.rg_count -1][0]]) : {
    name = "rt-from-${b.name}-to-hub"
    vnet_name = b.vnet_name
    subnet_name = b.name

    route = [for a in range(2) : { 
      name = a == 0 ? "all-internet-traffic-from-subnet-to-hub-first" : "all-internal-traffic-from-subnet-to-hub-first"
      address_prefix = a == 0 ? "0.0.0.0/0" : b.address_prefix[0]
      next_hop_type = "VirtualAppliance"
      next_hop_in_ip_address = local.tp_object.hub_object.network.fw_resource_id != null ? data.azurerm_firewall.firewall_hub_object[0].ip_configuration[0].private_ip_address : local.tp_object.hub_object.network.fw_private_ip != null ? local.tp_object.hub_object.network.fw_private_ip : local.fw_return_helper_object[0].ip_configuration[0].private_ip_address
    }]
  }] : []

  ####################################################
  ###### VARIABLE OBJECTS TRANSFORMATION TO MAPS #####
  ####################################################
  
  gw_object = local.tp_object.hub_object.network == null ? {} : local.tp_object.hub_object.network.vpn != null || local.tp_object.hub_object.network.vpn == {} ? {for each in [{
    name = replace(replace(local.tp_object.hub_object.network.vpn.gw_name == null ? local.gateway_base_name : local.tp_object.hub_object.network.vpn.gw_name, "/|(^-)|(-$)/", ""), "--", "-")
    vnet_name = [for a,b in local.vnet_objects_pre : b.name if a == local.rg_count -1][0]
    sku = local.tp_object.hub_object.network.vpn.gw_sku == null ? local.vpn_gateway_sku : local.tp_object.hub_object.network.vpn.gw_sku
    tags = merge(local.tp_object.tags, local.tp_object.hub_object.network.tags, local.tp_object.hub_object.network.vpn.tags)
    type = "Vpn"
    remote_vnet_traffic_enabled = true
    generation = "Generation2"
    private_ip_address_enabled = true

    ip_configuration = {
      subnet_id = [for a, b in local.subnet_return_helper_objects : b.id if b.name == "GatewaySubnet"][0]
    }

    vpn_client_configuration = {
      address_space = local.tp_object.hub_object.network.vpn.address_space != null ? local.tp_object.hub_object.network.vpn.address_space : tonumber(split(".", [for a, b in local.vnet_objects_pre : b.address_spaces[0] if a == local.rg_count -1][0])[0]) == 10 ? ["172.16.99.0/24"] : ["10.99.0.0/24"]
      aad_tenant = "https://login.microsoftonline.com/${local.tenant_id}/"
      aad_issuer = "https://sts.windows.net/${local.tenant_id}/"
      vpn_client_protocols = ["OpenVPN"]
      vpn_auth_types = ["AAD"]
    }
  }] : each.name => each} : {}

  pip_objects_pre = local.pip_count == 2 ? [for a, b in range(local.pip_count) : {
      name = replace(a == 1 && local.tp_object.hub_object.network.vpn.pip_name != null ? local.tp_object.hub_object.network.vpn.pip_name : a == 1 && local.tp_object.hub_object.network.vpn.pip_name == null ? replace(local.gateway_base_name, "gw", "gw-pip") : a == 0 && local.tp_object.hub_object.network.firewall.pip_name == null ? replace(local.gateway_base_name, "gw", "fw-pip") : local.tp_object.hub_object.network.firewall.pip_name, "/(--)|(^-)|(-$)/", "")
      vnet_name = [for e, f in local.vnet_objects_pre : f.name if e == local.rg_count -1][0]
      tags = merge(local.tp_object.tags, local.tp_object.hub_object.network.tags)
      ddos_protection_mode = null
      sku = "Standard"
      sku_tier = "Regional"
      allocation_method = "Static"
    }
  ] : []

  pip_objects_pre_2 = local.pip_count < 2 ? [for a, b in range(local.pip_count) : {
      name = replace([for c, d in [local.tp_object.hub_object.network.vpn, local.tp_object.hub_object.network.firewall] : d if d != null][0].pip_name != null ? [for c, d in [local.tp_object.hub_object.network.vpn, local.tp_object.hub_object.network.firewall] : d if d != null][0].pip_name : local.tp_object.hub_object.network.vpn != null ? replace(local.gateway_base_name, "gw", "gw-pip") : replace(local.gateway_base_name, "gw", "fw-pip"), "/(--)|(^-)|(-$)/", "")
      vnet_name = [for e, f in local.vnet_objects_pre : f.name if e == local.rg_count -1][0]
      tags = merge(local.tp_object.tags, local.tp_object.hub_object.network.tags)
      ddos_protection_mode = null
      sku = "Standard"
      sku_tier = "Regional"
      allocation_method = "Static"
    }
  ] : []

  fw_object = !can(local.tp_object.hub_object.network.firewall) ? {} : local.tp_object.hub_object.network.firewall != null ? {for each in [for a, b in range(1) : {
    name = replace(replace(local.tp_object.hub_object.network.firewall.name != null ? local.tp_object.hub_object.network.firewall.name : replace(local.gateway_base_name, "gw", "fw"), "/|(^-)|(-$)/", ""), "--", "-")
    sku_name = local.wan_object == {} ? "AZFW_VNet" : "AZFW_Hub"
    sku_tier = local.tp_object.hub_object.network.firewall.sku_tier != null ? local.tp_object.hub_object.network.firewall.sku_tier : "Standard"
    threat_intel_mode = local.tp_object.hub_object.network.firewall.threat_intel_mode != null ? "Deny" : "Alert"
    vnet_name = [for c , d in local.vnet_objects_pre : d.name if c == local.rg_count -1][0]
    tags = merge(local.tp_object.tags, local.tp_object.hub_object.network.tags, local.tp_object.hub_object.network.firewall.tags)

    ip_configuration = {
      name = "fw-config"
      subnet_id = [for c, d in local.subnet_return_helper_objects : d.id if d.name == "AzureFirewallSubnet"][0]
    }

    virtual_hub = local.wan_object == {} ? {} : {for each in [
      {
        virtual_hub_id = null
      }
    ] : each.virtual_hub_id => each}
  }] : each.name => each} : {}

  fw_log_object = !can(local.tp_object.hub_object.network.firewall.no_logs) ? {} : local.tp_object.hub_object.network.firewall.no_logs == null ?  {for each in [for c, d in range(1) : {
    name = replace(replace(local.tp_object.hub_object.network.firewall.log_name != null ? local.tp_object.hub_object.network.firewall.log_name : replace(local.gateway_base_name, "gw", "log-fw"), "/(^-)|(-$)/", ""), "--", "-")
    daily_quota_gb = local.tp_object.hub_object.network.firewall.log_daily_quota_gb
    tags = merge(local.tp_object.tags, local.tp_object.hub_object.network.tags)
  }] : each.name => each} : {}

  fw_diag_object = !can(local.tp_object.hub_object.network.firewall.no_logs) ? {} : local.tp_object.hub_object.network.firewall.no_logs == null ? {for each in [for c, d in range(1) : {
    name = local.tp_object.hub_object.network.firewall.log_diag_name != null ? local.tp_object.hub_object.network.firewall.log_diag_name : "fw-logs-to-log-analytics"
    log_analytics_destination_type = "Dedicated" #Static
    category_group = "AllLogs" #Static
  }] : each.name => each} : {}

  fw_rule_objects_pre = !can(local.tp_object.hub_object.network.firewall.no_rules) ? {} : local.tp_object.hub_object.network.firewall.no_rules == null ? {for each in [for a, b in range(2) : { #Must be by itself so that the rule ONLY relies on the GW finishing deploying and not the FW
      name = a == 0 ? "Allow-HTTP-HTTPS-DNS-FROM-SPOKES-TO-INTERNET" : "Allow-RDP-SSH-FROM-MGMT-TO-SPOKES"
      priority = a == 0 ? 100 : 200
      action = "Allow"
      source_addresses = a == 0 && local.tp_object.hub_object.network.vnet_spoke_address_spaces != null ? local.tp_object.hub_object.network.vnet_spoke_address_spaces : a == 0 ? flatten([for c, d in local.vnet_objects_pre : d.address_spaces if d.name != [for e, f in local.vnet_objects_pre : f.name if e == local.rg_count -1][0]]) : can([for c, d in values(local.subnet_objects) : d.address_prefix if length(regexall("mgmt|management", lower(d.name))) > 0][0]) ? [for c, d in values(local.subnet_objects) : d.address_prefix if length(regexall("mgmt|management", lower(d.name))) > 0][0] : flatten([for c, d in local.vnet_objects_pre : d.address_spaces if d.name == [for e, f in local.vnet_objects_pre : f.name if e == local.rg_count -1][0]])
      destination_ports = a == 0 ? ["53", "80", "443"] : ["22", "3389"]
      destination_addresses = a == 0 ? ["0.0.0.0/0"] : local.tp_object.hub_object.network.vnet_spoke_address_spaces != null ? local.tp_object.hub_object.network.vnet_spoke_address_spaces : flatten([for c, d in local.vnet_objects_pre : d.address_spaces if d.name != [for e, f in local.vnet_objects_pre : f.name if e == local.rg_count -1][0]])
      protocols = a == 0 ? ["TCP", "UDP"] : ["TCP"]
      vnet_name = [for c, d in local.vnet_objects_pre : d.name if c == local.rg_count -1][0]
  }] : each.name => each} : {}

  //MARK: SEPERATING HUB - SPOKES
  
  rg_hub_object = local.tp_object.hub_object.network.vnet_resource_id == null ? {for each in [values(local.rg_objects)[0]] : each.name => each} : {}
  rg_spoke_objects = can(length(local.tp_object.spoke_objects)) ? {for each in flatten([for a, b in values(local.rg_objects) : b if b.name != values(local.rg_objects)[0].name]) : each.name => each} : {}
  vnet_hub_object = local.tp_object.hub_object.network.vnet_resource_id == null ? {for each in flatten([for a, b in local.vnet_objects_pre : b if a == local.rg_count -1]) : each.name => each} : {}
  vnet_spoke_objects = can(length(local.tp_object.spoke_objects)) ? {for each in flatten([for a, b in local.vnet_objects_pre : b if a != local.rg_count -1]) : each.name => each} : {}
  subnet_objects = {for each in (flatten(local.subnet_objects_pre.*.subnets)) : each.name_unique => each}
  subnet_hub_objects = {for each in [for a, b in (flatten(local.subnet_objects_pre.*.subnets)) : b if b.vnet_name == [for c, d in local.vnet_objects_pre : d.name if c == local.rg_count -1][0]] : each.name_unique => each}
  subnet_spoke_objects = {for each in [for a, b in (flatten(local.subnet_objects_pre.*.subnets)) : b if b.vnet_name != [for c, d in local.vnet_objects_pre : d.name if c == local.rg_count -1][0]] : each.name_unique => each}
  peering_hub_objects = {for each in local.peering_objects_from_hub_to_spokes : each.name => each }
  peering_spoke_objects = {for each in local.peering_objects_from_spokes_to_hub : each.name => each }
  route_table_objects = {for each in local.route_table_objects_pre : each.name => each}
  pip_objects = {for each in [for a, b in flatten([local.pip_objects_pre, local.pip_objects_pre_2]) : b if b != []] : each.name => each}
  fw_rule_objects = !can(local.tp_object.spoke_objects[0].network.subnet_objects[0]) && local.tp_object.hub_object.network.vnet_spoke_address_spaces == null ? {} : !can(local.tp_object.hub_object.network.firewall.no_internet) ? {} : local.tp_object.hub_object.network.firewall.no_internet != null && local.tp_object.hub_object.network.vnet_spoke_address_spaces != null || local.tp_object.hub_object.network.firewall.no_internet != null && can(local.tp_object.spoke_objects[0].network.subnet_objects[0]) ? {for each in [values(local.fw_rule_objects_pre)[1]] : each.name => each} : !can(local.tp_object.hub_object.network.firewall.no_internet) ? {} : local.fw_rule_objects_pre

  ############################################
  ########## VARIABLE RETURN OBJECTS #########
  ############################################

  //MARK: RETURN VARIABLES

  rg_return_helper_objects = local.rg_return_objects != {} ? values(local.rg_return_objects) : []
  rg_return_objects = merge(azurerm_resource_group.rg_hub_object, azurerm_resource_group.rg_spoke_object)
  vnet_return_objects = merge(azurerm_virtual_network.vnet_hub_object, azurerm_virtual_network.vnet_spoke_object)
  vnet_return_helper_objects = local.vnet_return_objects != {} ? values(local.vnet_return_objects) : []
  subnet_return_objects = merge(azurerm_subnet.subnet_hub_object, azurerm_subnet.subnet_spoke_object)
  subnet_return_helper_objects = local.subnet_return_objects != {} ? values(local.subnet_return_objects) : []
  peering_return_objects = merge(azurerm_virtual_network_peering.peering_hub_object, azurerm_virtual_network_peering.peering_spoke_object)
  pip_return_object = azurerm_public_ip.pip_object
  pip_return_helper_objects = azurerm_public_ip.pip_object != {} ? values(azurerm_public_ip.pip_object) : []
  gw_return_object = azurerm_virtual_network_gateway.gw_vpn_object
  gw_return_helper_object = azurerm_virtual_network_gateway.gw_vpn_object != {} ? values(azurerm_virtual_network_gateway.gw_vpn_object) : []
  rt_return_objects = azurerm_route_table.route_table_from_spokes_to_hub_object
  rt_return_helper_objects = azurerm_route_table.route_table_from_spokes_to_hub_object != {} ? values(azurerm_route_table.route_table_from_spokes_to_hub_object) : []
  fw_return_helper_object = azurerm_firewall.fw_object != {} ? values(azurerm_firewall.fw_object) : []
  fw_return_object = azurerm_firewall.fw_object
  log_return_object = azurerm_log_analytics_workspace.fw_log_object
  log_return_helper_object = azurerm_log_analytics_workspace.fw_log_object != {} ? values(azurerm_log_analytics_workspace.fw_log_object) : []
} 

  ############################################
  ############ DATA DEFINITIONS ##############
  ############################################

//MARK: USER CONTEXT HUB

data "azurerm_client_config" "context_object"{
  count = local.create_vpn ? 1 : 0
  provider = azurerm.hub
}

//MARK: USER CONTEXT SPOKE
data "azurerm_client_config" "context_spoke_object" {
  count = can(length(local.tp_object.spoke_objects)) ? 1 : 0

  provider = azurerm.spoke
}

//MARK: FW PRIVATE IP
data "azurerm_firewall" "firewall_hub_object" {
  count = local.tp_object.hub_object.network.fw_resource_id != null ? 1 : 0
  resource_group_name = split("/", local.tp_object.hub_object.network.fw_resource_id)[4]
  name = split("/", local.tp_object.hub_object.network.fw_resource_id)[8]

  provider = azurerm.hub
}

  ############################################
  ############ RESOURCE DEFINITIONS ##########
  ############################################


//MARK: RESOURCE GROUP - HUB
resource "azurerm_resource_group" "rg_hub_object" { 
  for_each = local.rg_hub_object
  name = each.key
  location = each.value.location
  tags = each.value.tags

  provider = azurerm.hub
}

//MARK: RESOURCE GROUP´S - SPOKES
resource "azurerm_resource_group" "rg_spoke_object" {
  for_each = local.rg_spoke_objects
  name = each.key
  location = each.value.location
  tags = each.value.tags

  provider = azurerm.spoke
}

//MARK: VNET - HUB
resource "azurerm_virtual_network" "vnet_hub_object" {
  for_each = local.vnet_hub_object
  name = each.key
  location = [for a in local.rg_objects : a.location if a.vnet_name == each.key][0]
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.key][0]
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

  depends_on = [ azurerm_resource_group.rg_hub_object]

  provider = azurerm.hub
}

//MARK: VNET´S - SPOKES
resource "azurerm_virtual_network" "vnet_spoke_object" {
  for_each = local.vnet_spoke_objects
  name = each.key
  location = [for a in local.rg_objects : a.location if a.vnet_name == each.key][0]
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.key][0]
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

  depends_on = [ azurerm_resource_group.rg_spoke_object ]

  provider = azurerm.spoke
}

//MARK: vWAN (NOT DONE)

resource "azurerm_virtual_wan" "wan_object" {
  for_each = local.wan_object
  name = each.key
  location = each.value.location
  resource_group_name = each.value.resource_group_name
}

//MARK: SUBNETS - HUB
resource "azurerm_subnet" "subnet_hub_object" {
  for_each = local.subnet_hub_objects
  name = each.value.name
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0]
  virtual_network_name = each.value.vnet_name
  address_prefixes = each.value.address_prefix
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
  
  depends_on = [ azurerm_virtual_network.vnet_hub_object ]

  provider = azurerm.hub
}

//MARK: SUBNETS - SPOKES
resource "azurerm_subnet" "subnet_spoke_object" {
  for_each = local.subnet_spoke_objects
  name = each.value.name
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0]
  virtual_network_name = each.value.vnet_name
  address_prefixes = each.value.address_prefix
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
  
  depends_on = [ azurerm_virtual_network.vnet_spoke_object ]

  provider = azurerm.spoke
}

//MARK: PEERINGS - HUB
resource "azurerm_virtual_network_peering" "peering_hub_object" {
  for_each = local.peering_hub_objects
  name = each.key
  virtual_network_name = each.value.vnet_name != null ? each.value.vnet_name : split("/", local.tp_object.hub_object.network.vnet_resource_id)[8]
  remote_virtual_network_id = each.value.remote_virtual_network_id
  resource_group_name = can([for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0]) ? [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0] : split("/", local.tp_object.hub_object.network.vnet_resource_id)[4]
  allow_virtual_network_access = each.value.allow_virtual_network_access
  allow_forwarded_traffic = each.value.allow_forwarded_traffic
  allow_gateway_transit = each.value.allow_gateway_transit
  use_remote_gateways = each.value.use_remote_gateways

  depends_on = [ azurerm_virtual_network.vnet_hub_object,  azurerm_virtual_network.vnet_spoke_object]

  provider = azurerm.hub
}

//MARK: PEERINGS - SPOKES
resource "azurerm_virtual_network_peering" "peering_spoke_object" {
  for_each = local.peering_spoke_objects
  name = each.key
  virtual_network_name = each.value.vnet_name
  remote_virtual_network_id = each.value.remote_virtual_network_id
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0]
  allow_virtual_network_access = each.value.allow_virtual_network_access
  allow_forwarded_traffic = each.value.allow_forwarded_traffic
  allow_gateway_transit = each.value.allow_gateway_transit
  use_remote_gateways = each.value.use_remote_gateways

  depends_on = [ azurerm_virtual_network.vnet_hub_object,  azurerm_virtual_network.vnet_spoke_object]

  provider = azurerm.spoke
}

//MARK: ROUTE TABLES - SPOKES
resource "azurerm_route_table" "route_table_from_spokes_to_hub_object" {
  for_each = local.route_table_objects
  name = each.value.name
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0]
  location = [for a in local.rg_objects : a.location if a.vnet_name == each.value.vnet_name][0]
  route = each.value.route
  
  depends_on = [ azurerm_resource_group.rg_spoke_object ]

  provider = azurerm.spoke
}

//MARK: ROUTE ASSO - SPOKES
resource "azurerm_subnet_route_table_association" "link_route_table_to_subnet_object" {
  for_each = local.route_table_objects
  route_table_id = [for a, b in local.rt_return_helper_objects : b.id if b.name == each.key][0]
  subnet_id = [for a, b in local.subnet_return_helper_objects : b.id if b.name == each.value.subnet_name][0]

  provider = azurerm.spoke
}

//MARK: PUBLIC IP´S - HUB
resource "azurerm_public_ip" "pip_object" {
  for_each = local.pip_objects
  name = each.key
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0]
  location = [for a in local.rg_objects : a.location if a.vnet_name == each.value.vnet_name][0]
  sku = each.value.sku
  sku_tier = each.value.sku_tier
  allocation_method = each.value.allocation_method
  tags = each.value.tags

  depends_on = [ azurerm_resource_group.rg_hub_object]

  provider = azurerm.hub
}

//MARK: VPN - HUB
resource "azurerm_virtual_network_gateway" "gw_vpn_object" {
  for_each = local.gw_object
  name = each.key
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0]
  location = [for a in local.rg_objects : a.location if a.vnet_name == each.value.vnet_name][0]
  sku = each.value.sku
  type = each.value.type
  generation = each.value.generation
  private_ip_address_enabled = each.value.private_ip_address_enabled
  remote_vnet_traffic_enabled = each.value.remote_vnet_traffic_enabled
  tags = each.value.tags
  
  ip_configuration {
    subnet_id = each.value.ip_configuration.subnet_id
    public_ip_address_id = local.pip_count == 2 ? local.pip_return_helper_objects[1].id : local.pip_return_helper_objects[0].id
  }

  vpn_client_configuration {
    address_space = each.value.vpn_client_configuration.address_space
    aad_tenant = each.value.vpn_client_configuration.aad_tenant
    aad_audience = "41b23e61-6c1e-4545-b367-cd054e0ed4b4" #WIll always be this ID for the application Azure VPN
    aad_issuer = each.value.vpn_client_configuration.aad_issuer
    vpn_client_protocols = each.value.vpn_client_configuration.vpn_client_protocols
    vpn_auth_types = each.value.vpn_client_configuration.vpn_auth_types
  }

  depends_on = [ azurerm_resource_group.rg_hub_object]

  provider = azurerm.hub
}

//MARK: FIREWALL - HUB
resource "azurerm_firewall" "fw_object" {
  for_each = local.fw_object
  name = each.key
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0]
  location = [for a in local.rg_objects : a.location if a.vnet_name == each.value.vnet_name][0]
  sku_name = each.value.sku_name
  sku_tier = each.value.sku_tier
  threat_intel_mode = each.value.threat_intel_mode
  tags = each.value.tags 
  
  ip_configuration {
    name = each.value.ip_configuration.name
    subnet_id = each.value.ip_configuration.subnet_id
    public_ip_address_id = [for a, b in local.pip_return_helper_objects : b.id if b.name == [for c, d in values(local.pip_objects) : d.name if d.vnet_name == each.value.vnet_name][0]][0]
  }

  dynamic "virtual_hub" {
    for_each = each.value.virtual_hub
    content {
      virtual_hub_id = virtual_hub.key
    }
  }

  depends_on = [ azurerm_resource_group.rg_hub_object]

  provider = azurerm.hub
}

//MARK: FIREWALL RULE´S - HUB
resource "azurerm_firewall_network_rule_collection" "fw_rule_object" {
  for_each = local.fw_rule_objects
  name = each.key
  azure_firewall_name = local.fw_return_helper_object[0].name
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.value.vnet_name][0]
  priority = each.value.priority
  action = each.value.action

  rule {
    name = each.key
    source_addresses = each.value.source_addresses
    destination_addresses = each.value.destination_addresses
    destination_ports = each.value.destination_ports
    protocols = each.value.protocols
  }

  depends_on = [ azurerm_firewall.fw_object ]

  provider = azurerm.hub
}

//MARK: LOGSPACE - HUB
resource "azurerm_log_analytics_workspace" "fw_log_object" {
  for_each = local.fw_log_object
  name = each.key
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == [for b, c in local.vnet_objects_pre : c.name if b == local.rg_count -1][0]][0]
  location = [for a in local.rg_objects : a.location if a.vnet_name == [for b, c in local.vnet_objects_pre : c.name if b == local.rg_count -1][0]][0]
  daily_quota_gb = each.value.daily_quota_gb
  tags = each.value.tags

  depends_on = [ azurerm_firewall.fw_object ]

  provider = azurerm.hub
}

//MARK: DIAG SETTINGS - HUB
resource "azurerm_monitor_diagnostic_setting" "fw_diag_object" {
  for_each = local.fw_diag_object
  name = each.key
  log_analytics_destination_type = each.value.log_analytics_destination_type
  log_analytics_workspace_id = local.log_return_helper_object[0].id
  target_resource_id = local.fw_return_helper_object[0].id

  enabled_log {
    category_group = each.value.category_group
  }

  depends_on = [ azurerm_firewall.fw_object ]

  provider = azurerm.hub
}