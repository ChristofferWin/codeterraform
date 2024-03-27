  run "integration_test_1_create_apply" {
    command = apply //We want to create the actual resources  

    module {
      source = "./pre-deployment"
    }
  }

  run "integration_test_2_ids_apply" {
    command = apply //We want to create the actual resources

    //We tell the test framework to ONLY run on module 2
    //Otherwise each test would run the entire "integration_tests.tf"

    variables {
      //Location is defined in variables.tf
      //All 4 ids below are directly pulled via the outputs.tf file
      rg_id = run.integration_test_1_create_apply.rg_id
      vnet_resource_id = run.integration_test_1_create_apply.vnet_resource_id
      subnet_resource_id = run.integration_test_1_create_apply.subnet_resource_id
      kv_resource_id = run.integration_test_1_create_apply.kv_resource_id

      //Overwriting the default values of vms as the names must be unique in the same RG
      //This means even though the same input variable names are defined in both module calls
      //within the "integration_tests.tf" For this test 2, both input variables has their default values overwritten
      vm_windows_objects = [
        {
          name = "win-vm-02"
          os_name = "server2019"
        }
      ]

      vm_linux_objects = [
        {
          name = "linux-vm-02"
          os_name = "ubuntu"
        }
      ]
    }
  }