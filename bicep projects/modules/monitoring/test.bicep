param metricAlerts_HTTP_SERVER_ERROR_HIGH_name string = 'HTTP-SERVER-ERROR-HIGH'
param sites_tester1337_externalid string = '/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test-rg/providers/Microsoft.Web/sites/tester1337'
param actiongroups_test_ag_externalid string = '/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test-rg/providers/microsoft.insights/actiongroups/test-ag'

resource metricAlerts_HTTP_SERVER_ERROR_HIGH_name_resource 'microsoft.insights/metricAlerts@2018-03-01' = {
  name: metricAlerts_HTTP_SERVER_ERROR_HIGH_name
  location: 'global'
  properties: {
    severity: 1
    enabled: true
    scopes: [
      sites_tester1337_externalid
    ]
    evaluationFrequency: 'PT1M'
    windowSize: 'PT5M'
    criteria: {
      allOf: [
        {
          threshold: 10
          name: 'Metric1'
          metricNamespace: 'Microsoft.Web/sites'
          metricName: 'HealthCheckStatus'
          operator: 'LessThan'
          timeAggregation: 'Average'
          skipMetricValidation: false
          criterionType: 'StaticThresholdCriterion'
        }
      ]
      'odata.type': 'Microsoft.Azure.Monitor.SingleResourceMultipleMetricCriteria'
    }
    autoMitigate: true
    targetResourceType: 'Microsoft.Web/sites'
    targetResourceRegion: 'westeurope'
    actions: [
      {
        actionGroupId: actiongroups_test_ag_externalid
        webHookProperties: {}
      }
    ]
  }
}