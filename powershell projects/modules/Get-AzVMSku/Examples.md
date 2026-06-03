# Table of Contents

* [Description](#description)
* [Return object schema](#return-object-schema)
* [Examples](#examples)

  * [Example 1 - Using the different show-commands](#example-1---using-the-different-show-commands)
  * [Example 2 - Browse the Azure Image Marketplace](#example-2---browse-the-azure-image-marketplace)
  * [Example 3 - Browsing, but with filters](#example-3---browsing-but-with-filters)
  * [Example 4 - Use module in CI / CD pipelines](#example-4---use-module-in-ci--cd-pipelines)
  * [Example 5 - Handling VMs](#example-5---handling-vms)
  * [Example 6 - Accepting Image Marketplace terms](#example-6---accepting-image-marketplace-terms)

---

## Description

This markdown showcases different examples of how to use the PowerShell module Get-AzVMSku. One very important factor to note is these examples ONLY showcase using major version 3.0.0 and up - Before this version the module worked very differently and could only serve within statically typed operating systems which now has changed drastically.

<b>USING PARAMETER 'Verbose' IS ALWAYS RECOMMENDED</b>

PLEASE NOTE:

1. Vendors available will vary depending on location chosen
2. Since Microsoft doesn't clean up the gallery very well, you might pick an offer or later SKU where no actual version exists. In this case the module will throw a warning and return to the choices of vendors.
3. We can define image filters using:

   1. PublisherName

      * Searches using exact match first, then wildcard matching (*PublisherName*)
   2. PublisherNameStartsWith

      * Searches for publishers whose names begin with the provided value
   3. OfferName

      * Filters offers within the selected publisher
   4. NewestSKUs

      * Automatically selects the newest SKU
   5. NewestSKUsVersions

      * Automatically selects the newest image version
   6. UnfilteredPublishers

      * By default the module removes publishers containing:

        * test
        * .
        * large numeric identifiers
      * Use this switch to include all publishers exactly as returned by Azure

<b>Please see the examples for more details on the above.</b>

[⬆️ Back to Table of Contents](#table-of-contents)

---

## Return object schema

```ps1

# ROOT OBJECT
[PSCustomObject]@{
    Context = [PSCustomObject]@{
        SubscriptionID = ""
        SubscriptionName = ""
        TenantID = ""
        TenantName = ""
    }

    Publisher = ""
    Offer = ""
    SKU = ""
    Version = ""
    URN = ""

    VMs = @()

    VMSizePattern = ""

    Agreement = [Microsoft.Azure.Management.Compute.Models.PurchasePlan]
}

# CONTEXT
[PSCustomObject]@{
    SubscriptionID = ""
    SubscriptionName = ""
    TenantID = ""
    TenantName = ""
}

# VMs
[PSCustomObject]@{
    FamilyName = ""
    Status = ""
    AvailablevCPUQuota = 0
    CPUQuotaConsumedPercent = ""
    RemainingVMCapacity = @()
    Sizes = @()
    VMsCanBeDeployed = $false
}

# RemainingVMCapacity
[PSCustomObject]@{
    SizeName = ""
    VMCountBeforeQuotaLimit = 0
}

# Sizes
[PSCustomObject]@{
    Name = ""
    CoresAvailable = 0
    MemoryInGB = 0
    MaxDataDiskCount = 0
    HyperVGeneration = ""
    MaxNetworkInterfaces = 0
    RetirementDate = ""
}
```

### VM Status interpretation

* AvailablevCPUQuota = Remaining quota available for the VM family
* CPUQuotaConsumedPercent = Current quota consumption percentage
* VMsCanBeDeployed = Indicates whether deployment is currently possible
* RemainingVMCapacity = Number of VMs that can be created before quota exhaustion
* Sizes = Detailed VM specifications for each supported size
* Status = Human-readable deployment guidance

[⬆️ Back to Table of Contents](#table-of-contents)

---

## Examples

Below you can see examples showcasing all the different scenarios the module can compute - Please make sure to read the comment-blocks as they show intent.

---

### Example 1 - Using the different show-commands

The module can serve information about all possible Azure locations which the module accepts + It can print descriptions about any Azure Virtual Machine family directly from Microsoft.

<b>SEE AVAILABLE LOCATIONS</b>

```ps1
<#You must first be logged into a valid Azure context
Using either Login-AzAccount or Connect-AzAccount
#>

Get-AzVMSku -ShowLocations

<# PARTIAL OUTPUT
Name to use        Display name
-----------        ------------
malaysiawest       Malaysia West
indonesiacentral   Indonesia Central
chilecentral       Chile Central
australiacentral   Australia Central
......
#>
```

<b>SEE AZURE VM FAMILY DESCRIPTIONS</b>

```ps1
#You can run this WITHOUT Azure context
Get-AzVMSku -ShowVMCategories

<# PARTIAL OUTPUT
Title      Description
-----      -----------
A-Series   A-series VMs have CPU performance and memory configurations
Bs-Series  Bs-series VMs are economical virtual machines
D-Series   The D-series Azure VMs offer a combination of vCPUs, memory, and temporary storag…
............
#>
```

[⬆️ Back to Table of Contents](#table-of-contents)

---

### Example 2 - Browse the Azure Image Marketplace

The module allows for a full browsing experience without knowing ANYTHING about any specific vendor, what they offer and so on. The only thing we need to provide as a base, is the location.

<b>USE ONLY LOCATION TO BROWSE</b>

```ps1
#We MUST have an active Azure context first
Get-AzVMSku -Location westeurope

<# PARTIAL OUTPUT OF OVER 2300 vendors
###### Please select a specific Image Publisher below ######
[1] - 100101010000
[2] - 128technology
[3] - 1580863854728
[4] - 1583411303229
..................
[2373] - zscaler1579058425289
[2374] - zultysinc1596831546163
Your selection...:
#>

<#
We can now select a specific vendor and depending on the vendor and offer chosen we might have to:

1. Select Image SKU from a list
2. Select the Image SKU version

The above will completely depend on the selections we make and in case any given SKU / Version only has 1 option, the module will auto-select it and move on.
#>
```

[⬆️ Back to Table of Contents](#table-of-contents)

---

### Example 3 - Browsing, but with filters

More than simply browsing everything that's available - We can add filters to our image search to make the returned results more bearable.

<b>USING PUBLISHERNAME</b>

```ps1
#Looking for something to-do with palo
Get-AzVMSku -Location westeurope -PublisherName palo

<# PARTIAL OUTPUT
1 match found for PublisherName 'palo' => paloaltonetworks

###### Please select a specific Image Offer below ######
[1] - airs-flex
[2] - cortex_xsoar
[3] - pan-prisma-access-ztna-connector
[4] - pan-prisma-access-ztna-fedramp-connector
.........
[10] - vmseries-forms
[11] - vmseries1
[12] - vwan-managed-nva

Your selection...:
#>

<#
Notice how the module clearly states that it found 1 specific vendor from the filter 'palo' which lead to finding specific vendor 'paloaltonetworks'.

This also means that instead of having to choose vendor, it's already auto-selected as the only vendor found, and we can now focus on selecting an offer from said vendor.
#>
```

<b>USING PUBLISHERNAMESTARTSWITH</b>

```ps1
Get-AzVMSku -Location westeurope -PublisherNameStartsWith microsoft

This limits the search to publishers whose names begin with `microsoft`.

Examples:

* microsoftwindowsserver
* microsoftsqlserver
* microsoftcblmariner

This is useful when browsing Microsoft's marketplace images without knowing the exact publisher name.
```

<b>USING PUBLISHERNAME & OFFERNAME</b>

```ps1
#Adding more information to the example just before this one

Get-AzVMSku `
            -Location westeurope `
            -PublisherName palo `
            -OfferName flex

<# FULL OUTPUT
1 match found for PublisherName 'palo' => paloaltonetworks

###### Please select a specific Image Offer below ######
[1] - airs-flex
[2] - vmseries-flex

Your selection...:
#>

Leaving us with only flex offers from the vendor 'paloaltonetworks'
```

<b>INCLUDING ALL PUBLISHERS</b>

```ps1
Get-AzVMSku `
            -Location westeurope `
            -PublisherName microsoft `
            -UnfilteredPublishers
```

By default the module removes publishers that appear to be:

* test publishers
* publishers containing dots (.)
* publishers containing long numeric identifiers

This helps reduce noise when browsing Azure Marketplace.

Use `-UnfilteredPublishers` when you want the complete publisher list exactly as returned by Azure.

[⬆️ Back to Table of Contents](#table-of-contents)

---

### Example 4 - Use module in CI / CD pipelines

In pipelines / automation we cannot allow interactions as it will fail any automatic flow. To get around this, we can combine all filters and the parameter `NoInteractive` to make sure the module can run correctly without ANY user interactions.

<b>USING ALL FILTERS</b>

```ps1
<# 
Please note that this will require information from the user as we run the module.

If we build on from the example above, we now know PublisherName and a concrete OfferName to make sure only 1 result returns.

If we combine this information with:

1. NewestSKUs
2. NewestSKUsVersions
3. NoInteractive

The module can run without any user interaction.

Since OfferName 'flex' returns multiple offers, we use 'airs' instead.
#>

Get-AzVMSku `
            -Location westeurope `
            -PublisherName palo `
            -OfferName airs `
            -NewestSKUs `
            -NewestSKUsVersions `
            -NoInteractive `
            -Verbose

<# PARTIAL OUTPUT
VERBOSE: No exact match found for PublisherName 'palo' trying with wild-cards...
1 match found for PublisherName 'palo' => paloaltonetworks

VERBOSE: No exact match found for OfferName 'airs' trying with wild-cards...
1 match for OfferName 'airs-flex' found using wild-cards

WARNING: Purchase plan detected for image: paloaltonetworks:airs-flex:airs-byol:11.2.501

VERBOSE: You can use the command Set-AzVMSku to accept the Azure Market terms.

#######################################
######### Image definition ############
#######################################

Publisher: paloaltonetworks
Offer : airs-flex
SKU : airs-byol
Version : 11.2.501
URN : paloaltonetworks:airs-flex:airs-byol:11.2.501
Image agreement: True

#######################################
#>
```

[⬆️ Back to Table of Contents](#table-of-contents)

---

### Example 5 - Handling VMs

The module calculates Azure VM quota availability and groups all information by VM family.

The property `VMs` contains:

* Quota information
* Deployment eligibility
* Remaining capacity calculations
* Detailed VM specifications

This makes it possible to determine:

1. Whether VMs can be deployed
2. Remaining vCPU quota
3. How many VMs of a specific size can be created
4. Detailed specifications for each VM size

Please read the [⬆️ Return Object Schema](#return-object-schema) section for a better understanding of the VM-related properties.

<b>FILTER ON VM-SIZES</b>

```ps1
Get-AzVMSku `
            -Location westeurope `
            -VmPattern 'D'
```

Example VM family output:

```ps1
FamilyName              : Ddsv4-Family
Status                  : VMs in this family can be deployed. Check the RemainingVMCapacity property for details
AvailablevCPUQuota      : 10
CPUQuotaConsumedPercent : 50 %
VMsCanBeDeployed        : True
```

Example RemainingVMCapacity output:

```ps1
SizeName                 : Standard_D2a_v4
VMCountBeforeQuotaLimit  : 5

SizeName                 : Standard_D4a_v4
VMCountBeforeQuotaLimit  : 2
```

Example VM Size details:

```ps1
Name                 : Standard_D2a_v4
CoresAvailable       : 2
MemoryInGB           : 8
MaxDataDiskCount     : 4
HyperVGeneration     : V1,V2
MaxNetworkInterfaces : 2
RetirementDate       : No date found
```

<b>NO INFORMATION ABOUT VMs</b>

```ps1
#This will run MUCH faster
Get-AzVMSku `
            -Location westeurope `
            -NoVMInformation
```

```text
The output will skip all VM discovery and quota calculations.

The following properties will not contain VM data:

1. VMs
2. VMSizePattern

This significantly improves execution speed.
```

[⬆️ Back to Table of Contents](#table-of-contents)

---

### Example 6 - Accepting Image Marketplace terms

As it is with most Azure Images, specific Azure Image terms apply which we must accept BEFORE being able to deploy using an image.

We can do this seamlessly using the helper-function that is part of the module package.

<b>ACCEPTING MARKETPLACE TERMS</b>

```ps1
Get-AzVmSku -Location westeurope | Set-AzVMSku
```

The module will first print ALL specific terms to accept:

```text
[Privacy Policy => https://www.paloaltonetworks.com/legal/privacy.html]

[License => https://storeordersprodsn.blob.core.windows.net/legalterms/...txt]

[Marketplace Terms => https://storeordersprodsn.blob.core.windows.net/marketplaceterms/...txt]

Do you accept these terms? [y(yes)/n(no)]:
```

Accepting terms with `y` gives:

```text
The agreement for URN: paloaltonetworks:airs-flex:airs-byol:11.2.501 has been accepted and the image can be used in deployments
```

The image can now be used in any deployment on that specific Azure subscription.

<b>ACCEPTING TERMS WITH FORCE</b>

```ps1
#In automation scenarios, we want to skip confirmation

Get-AzVmSku -Location westeurope | Set-AzVMSku -Force
```

```text
Same output as above, but the module won't ask for confirmation.
```

[⬆️ Back to Table of Contents](#table-of-contents)
