targetScope = 'subscription'

@description('Provide a valid location, defaults to \'westeurope\'')
param location string = 'westeurope'

@description('Provide a resource group name')
param rg_name string

resource rg_object 'Microsoft.Resources/resourceGroups@2023-07-01' = {
  name: rg_name
  location: location
}

module storage_account '../storage_account/storage_account.bicep' = {
  name: 'test'
  scope: resourceGroup(rg_name)

  dependsOn: [rg_object]
}
