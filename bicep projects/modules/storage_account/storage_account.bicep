@description('Provide a valid environment, must be either \'prod\', \'test\' or \'dev\'')
@allowed(
  [
  'prod'
  'test'
  'dev'
  ]

)
param env_name string

@description('Provide a valid location, defaults to \'westeurope\'')
param location string = 'westeurope'

@description('To create a resource group for the storage account, use in case the param \'rg_name\' is set and the resource group is not created')
param create_rg bool = false

@description('Provide a valid resource group name defaults to env_name + random string of length 5')
param resource_group_name string = '${env_name}-${uniqueString('random')}-rg'

@description('Provide a storage account name, defaults to env_name + random string of length 5')
param storage_account_name string = '${env_name}${uniqueString('random')}storage'

@description('Provide a storage account sku, defaults to \'Standard_LRS\'')
param storage_account_sku sku = {
  name: 'Standard_LRS'
}

type sku = object

@description('Provide a storage account kind, defaults to \'BlobStorage\'')
@allowed([
  'BlobStorage'
  'BlockBlobStorage'
  'FileStorage'
  'Storage'
  'StorageV2'
])
param storage_account_kind string = 'BlobStorage'

module rg_object '../resource_group/resource_group.bicep' = if (create_rg) {
  scope: subscription()
  name: 'rg_object'
  params: {
    rg_name: resource_group_name
    location: location
  }
}

resource storage_object 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: storage_account_name
  sku: storage_account_sku
  kind: storage_account_kind
  location: location
  
  properties: {
    accessTier: 'Cool'
  }
}

output test string = resource_group_name 
