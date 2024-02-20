metadata name = 'Azure metric alerts'
metadata description = 'Deploy either default or customized monitoring for either Azure virtual machines or Azure function apps'
metadata owner = 'Christoffer Windahl Madsen @Codeterraform'

@description('Location of the resources deployed, defaults to location of resource group')
param location string = resourceGroup().location

@description('Provide any number of resource ids to add alerts for, supports virtual machines & function apps. If providing a custom alert config file, add the resource ids to that file instead of here')
param function_app_resource_ids array = [
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test/providers/Microsoft.Web/sites/test-1338'
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test/providers/Microsoft.Web/sites/test-1337'
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test/providers/Microsoft.Web/sites2/test-1338'
]

@description('Provide any number of resource ids to add alerts for, supports virtual machines & function apps. If providing a custom alert config file, add the resource ids to that file instead of here')
param virtual_machines_resource_ids array = [
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test/providers/Microsoft.Web/sites/test-1338'
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test/providers/Microsoft.Web/sites/test-1337'
'/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test/providers/Microsoft.Web/sites/test-1320'
]


@description('Provide a resource id of an already existing action group, alternatively supply the action group ids via the parameter \'alert_objects\'')
param action_group_resource_id string = '' //Optional

@description('Provide a list of emails to add to the default action group, alternatively supply the emails via the parameter \'action_group_objects\'')
param action_group_emails array = []

@description('Add an env name to be used as either prefix or suffix')
param env_name string = '' //Optional

@description('Switch to determine whether \'env_name shall\' be used as prefix or suffix, \'true\' = suffix, defaults to false')
param switch_env_name bool = false

@description('Switch to determine whether to simply deploy default alerts for the provided \'resource_ids\', defaults to true')
param default_alerts bool = true

@description('Log analytics resource id, in case a new one is not to be created')
param log_resource_id string = '' //Optional

@description('Log analytics custom name')
param log_resource_name string = 'metric_alerts.json' //Optional

@description('Path to valid JSON string representing alert objects, see the file \'metric_alerts.json\' for details')
param json_path string = '' //Optional

//Maybe we need to build the actual alert objects here already, since we cannot use can() like in terraform => Because of this we cannot check each property to see if an object contains it
var log_resource_name_final = !empty(log_resource_name) && !empty(env_name) && !switch_env_name ? '${env_name}-${log_resource_name}' : !empty(log_resource_name) && !empty(env_name) && switch_env_name ? '${log_resource_name}-${env_name}' : !empty(log_resource_name) && empty(env_name) ? log_resource_name : 'log-${take(uniqueString(resourceGroup().id), 5)}'
var metric_alerts_function_app_objects_template = [
  for each in range(0,3) : {
    name: each == 0 ? 'Default-HTTP-ERROR-HIGH' : each == 1 ? 'Default-SERVER-ERROR-HIGH' : 'Default-LATENCY-ERROR-HIGH'
      properties: {
        severity: 1
        enabled: true
        scopes: []
        evaluationFrequency: 'PT1M'
        windowSize: 'PT5M'
        autoMitigate: true
        targetResourceType: 'Microsoft.Web/sites'
        targetResourceRegion: location
        
        criteria: {
          allOf: [
            {
              threshold: each == 2 ? 10 : 100
              name: 'Metric1'
              metricNamespace: 'Microsoft.Web/sites'
              metricName: each == 0 ? 'HttpServerError' : each == 1 ? 'HealthCheckStatus' : 'ResponseTime'
              operator: 'LessThan'
              timeAggregation: 'Average'
              skipMetricValidation: false
              criterionType: 'StaticThresholdCriterion'
            }
          ]
          'odata.type': 'Microsoft.Azure.Monitor.SingleResourceMultipleMetricCriteria'
        }

        actions: [
          {
            actionGroupId: default_action_group_high_alerts.id
            webHookProperties: {}
          }
        ]
      }
  }
] 

var merge_default_function_app_alert_objects_temp = [for each in range(0, length(function_app_resource_ids)) : [
      function_app_resource_ids[each]
      metric_alerts_function_app_objects_template[0]
      metric_alerts_function_app_objects_template[1]
      metric_alerts_function_app_objects_template[2]
]]

var merge_function_app_alert_objects = flatten(merge_default_function_app_alert_objects_temp) //: //loadJsonContent('metric_alerts.json')

/*
//Will create a very simple logspace incase the param 'log_resource_id' is not provided. 90 days data retention
resource log_analytics_workspace 'Microsoft.OperationalInsights/workspaces@2022-10-01' = if(empty(log_resource_id)) {
  name: empty(log_resource_name) ? log_resource_name_final : log_resource_name
  location: location
}
*/
resource default_action_group_high_alerts 'Microsoft.Insights/actionGroups@2023-01-01' = if(empty(action_group_resource_id)) {
  name: 'ag-high'
  location: 'global'
  properties: {
    emailReceivers: action_group_emails
    enabled: !empty(action_group_emails) ? true : false
    groupShortName: 'aghigh'
  }
}

/*
resource metric_alerts_function_app 'Microsoft.Insights/metricAlerts@2018-03-01' = [for (each, i) in merge_function_app_alert_objects : {
  name: !empty(function_app_resource_ids) ? replace(each.name, 'Default', split(each.resource_id, '/')[8]) : each.name
  location: location
  properties: {
    enabled: each.properties.enabled
    autoMitigate: each.properties.autoMitigate
    evaluationFrequency: each.properties.evaluationFrequency
    scopes: [each.resource_id]
    windowSize: each.properties.windowSize
    severity: 1
    targetResourceType: 'Microsoft.Web/sites'
    targetResourceRegion: location

    criteria: {
      allOf: [
        {
          threshold: each == 2 ? 10 : 100
          name: 'Metric1'
          metricNamespace: 'Microsoft.Web/sites'
          metricName: each == 0 ? 'HttpServerError' : each == 1 ? 'HealthCheckStatus' : 'ResponseTime'
          operator: 'LessThan'
          timeAggregation: 'Average'
          skipMetricValidation: false
          criterionType: 'StaticThresholdCriterion'
        }
      ]
      'odata.type': each.properties.criteria.odata.type
    }
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
*/

output object int = length(merge_function_app_alert_objects[0])
