metadata name = 'Azure metric alerts'
metadata description = 'Deploy either default or customized monitoring for either Azure virtual machines or Azure function apps'
metadata owner = 'Christoffer Windahl Madsen @Codeterraform'

@description('Location of the resources deployed, defaults to location of resource group')
param location string = resourceGroup().location

@description('Provide any number of resource ids to add alerts for, supports virtual machines & function apps. If providing a custom alert config file, add the resource ids to that file instead of here')
param resource_ids array = [
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/RG-LKH/providers/Microsoft.Compute/virtualMachines/Win11dev'
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/RG-RightsManagementApp/providers/Microsoft.Compute/virtualMachines/HW-Automize-P01'
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/RG-RightsManagementApp/providers/Microsoft.Compute/tester/HW-Automize-P01'
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test/providers/Microsoft.Web/sites/test-1338'
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test/providers/Microsoft.Web/sites/test-1337'
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test/providers/Microsoft.Web/sites2/test-1338'
]

@description('Add an env name to be used as either prefix or suffix')
param env_name string = 'prod'

@description('Switch to determine whether \'env_name shall\' be used as prefix or suffix, \'true\' = suffix, defaults to false')
param switch_env_name bool = false

@description('Switch to determine whether to simply deploy default alerts for the provided \'resource_ids\', defaults to true')
param default_alerts bool = true

@description('Log analytics resource id, in case a new one is not to be created')
param log_resource_id string = ''

@description('Log analytics custom name')
param log_resource_name string = 'test-log'

//Maybe we need to build the actual alert objects here already, since we cannot use can() like in terraform => Because of this we cannot check each property to see if an object contains it
var log_resource_name_final = !empty(log_resource_name) && !empty(env_name) && !switch_env_name ? '${env_name}-${log_resource_name}' : !empty(log_resource_name) && !empty(env_name) && switch_env_name ? '${log_resource_name}-${env_name}' : !empty(log_resource_name) && empty(env_name) ? log_resource_name : 'log-${take(uniqueString(resourceGroup().id), 5)}'
var metric_alerts_resource_ids = empty(resource_ids) ? loadJsonContent('metric_alerts.json').resource_id : resource_ids
var metric_alerts_vm_resource_id = [for each in array(metric_alerts_resource_ids) : split(each, '/')[7] == 'virtualMachines' ? each : null]
var metric_alerts_function_app_resource_id = [for each in array(metric_alerts_resource_ids) : split(each, '/')[7] == 'sites' ? each : null]

resource metric_alerts_function_app 'Microsoft.Insights/metricAlerts@2018-03-01' = [for (each, i) in metric_alerts_function_app_resource_id: if (each != null) {
  name: 'lol'
  location: location

  properties: {
    criteria: ''
    enabled: ''
    evaluationFrequency: ''
    scopes: ''
    severity: ''
    windowSize: ''
  }
}]

resource metric_alerts_vm 'Microsoft.Insights/metricAlerts@2018-03-01' = [for (each, i) in metric_alerts_vm_resource_id: if (each != null) {
  name: ''
  location: location

  properties: {
    criteria: ''
    enabled: ''
    evaluationFrequency: '' 
    scopes: ''
    severity: ''
    windowSize: ''
  }
}]
