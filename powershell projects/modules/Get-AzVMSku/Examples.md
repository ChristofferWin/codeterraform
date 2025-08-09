# Table of Contents
- [Description](#description)
- [Return object schema](#return-object-schema)
- [Examples](#examples)
  - [Example 1 - Using the different show-commands](#example-1---using-the-different-show-commands)
  - [Example 2 - Browse the Azure Image Marketplace](#example-2---browse-the-azure-image-marketplace)
  - [Example 3 - Browsing, but with filters](#example-3---browsing-but-with-filters)
  - [Example 4 - Use module in CI / CD pipelines](#example-4---use-module-in-ci--cd-pipelines)
  - [Example 5 - Handling VMs](#example-5---handling-vms)
  - [Example 6 - Accepting Image Marketplace terms](#example-6---accepting-image-marketplace-terms)


## Description
This markdown showcases different examples of how to use the PowerShell module Get-AzVMSku. One very important factor to note is these examples ONLY showcase using major version 3.0.0 and up - Before this version the module worked very differently and could only serve within statically typed operating systems which now has changed drastically.

<b> USING PARAMETER 'Verbose' IS ALWAYS RECOMMENDED </b>

PLEASE NOTE:
1. Vendors available will vary depending on location chosen
2. Since Microsoft doesnt clean up the gallery very well, you might pick an offer or later SKU where no actual version exists - In this case the module will throw a warning and return to the choices of vendors
3. We can define name filters for both:
   1. Vendor-names via parameter 'PublisherName'
   2. Offer-names via parameter 'OfferName'
   3. We can auto-select newest SKU via parameter 'NewestSKUs
   4. We can auto-select newest SKU-version via parameter 'NewestSKUsVersions'

<b>Please see the examples for more details on the above.</b>

[⬆️ Back to Table of Contents](#table-of-contents)


## Return object schema
```ps1

#ROOT OBJECT
[PSCustomObject]@{
        SubscriptionID = ""
        SubscriptionName = ""
        TenantID = ""
        TenantName = ""
        Publisher = ""
        PublisherFilterName = ""
        Offer = ""
        OfferFilterName = ""
        SKU = ""
        NewestSku = ""
        URN = ""
        Version = ""
        NewestSkuVersion = ""
        VMSizes = [System.Collections.ArrayList]@()
        VMSizePattern = ""
        VmQuotas = [hashtable]
        Agreement = [Microsoft.Azure.Management.Compute.Models.PurchasePlan]
    }

#VMSizes ZOOM-IN (Captured as an array-list of [pscustomobject])
[PSCustomObject]@{
    Name =  ""
    CoresAvailable = 0
    MemoryInGB = 0
    MaxDataDiskCount = 0
    OSDiskSizeInGB = 0
    TempDriveSizeInGB = 0
}

#VMQuotas ZOOM-IN
@{
    "<VM-FAMILY-NAME>" = [PSCustomObject]@{
        vCPUsAvailable = 0
        ArchitectureDescription = ""
        vCPUsPercentUsage = 0
        VMSizeDistribution = [PSCustomObject]@{
            SizeName = ""
            VMCountBeforeLimit = 0
        }
    }
}

<#
The VM-Quota limit is a nested hash-table where data is stored about:

1. vCPUsAvailable if 0 = NO VMs can be deployed on the current subscripion for this VM-family. You MUST first ask Microsoft for a quota increase.

2. vCPUPercentUsage = How many % of my cores available is the Azure subscription using.

3. VMSizeDistribution = A PScustom object that calculates HOW many of any SPECIFIC vm-size under a SPECIFIC family can be created before the limit of the total Azure Subscription quota is reached.
#>
```
[⬆️ Back to Table of Contents](#table-of-contents)

## Examples
Below you can see examples showcasing all the different scenarios the module can compute - Please make sure to read the comment-blocks as they show intent.

### Example 1 - Using the different show-commands
The module can serve information about all possible Azure locations which the module accepts + It can print descriptions about any Azure Virtual machine family directly from Microsoft.

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

### Example 2 - Browse the Azure Image Marketplace
The module allows for a full browsing experience without knowing ANYTHING about any specific vendor, what they offer and so on. The only thing we need to provide as a base, is the location

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

The above will completly depend on the selections we make and in case any given SKU / Version only has 1 option, the module will auto-select it and move on.
#>
```
[⬆️ Back to Table of Contents](#table-of-contents)

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

Notice how the module clearly states that it found 1 specific vendor from the filter 'palo' Which lead to finding specific vendor 'paloaltonetworks'

This also means that instead of having to choose vendor, its already auto-selected as the only vendor found, and we can now focus on selecting an offer from said vendor.
#>
```

<b>USING PUBLISHERNAME & OFFERNAME</b>
```ps1
#Adding more information to the example just before this one - Now with 'palo' for PublisherName, we will add 'flex' For OfferName

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

Leaving us with only flex offers from the vendor 'paloaltonetworks'
#>
```
[⬆️ Back to Table of Contents](#table-of-contents)

### Example 4 - Use module in CI / CD pipelines
In pipelines / automation we cannot allow interactions as it will fail any automatic flow. To get around this, we can combine all filters and the parameter 'NoInteractive' To make sure the module can run correctly without ANY user-interactions.

<b>USING ALL FILTERS</b>
```ps1
<# Please note that this will require information from the user as we run the module. If we build on from the example 3 above, we now know PublisherName and a concrete Offername to make sure only 1 result returns. If we combine this information with the following parameters:

1. NewestSKUs
2. NewestSKUsVersions
3. NoInteractive

We can run the module and it wont ask for ANY user-input and return as per usual

Since OfferName 'flex' From example 3 returned 2 offers, we will use OfferName 'airs' Instead to force ONLY 1 offer
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
VERBOSE: You can use the command Set-AzVMSku to accept the Azure Market terms. Please note that deploying images with terms that are NOT accepted will lead to a failed deployment...
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

### Example 5 - Handling VMs
The module per default account for VM-sizes available in the current location / Azure region. To this it also calculates the VM-quotas available for the Azure subscription. With this we can:

1. Filter on VM-sizes, per default, the module will retrieve every single possible size and quota
2. Completly remove VM-information which makes the return MUCH faster

PLEASE read about the [⬆️ Return-Object schema](#return-object-schema) structure for a better understanding of the VM-related properties.

<b>FILTER ON VM-SIZES</b>
```ps1
Get-AzVMSku `
            -Location westeurope `
            -VmPattern 'D'

<# PARTIAL OUTPUT OF VM SIZES (Property VMSizes)
Name              : Standard_D2a_v4
CoresAvailable    : 2
MemoryInGB        : 8
MaxDataDiskCount  : 4
OSDiskSizeInGB    : 1023
TempDriveSizeInGB : 50

Name              : Standard_D4a_v4
CoresAvailable    : 4
MemoryInGB        : 16
MaxDataDiskCount  : 8
OSDiskSizeInGB    : 1023
TempDriveSizeInGB : 100
......................
Name              : Standard_DC48ds_v3
CoresAvailable    : 48
MemoryInGB        : 384
MaxDataDiskCount  : 32
OSDiskSizeInGB    : 1023
TempDriveSizeInGB : 2400

PARTIAL OUTPUT OF QUOTAS (Property VMQuotas)
Name                           Value
----                           -----
Ddsv4-Family                   @{vCPUsAvailable=10; ArchitectureDescription=The D-series Azu…
Dlsv5-Family                   @{vCPUsAvailable=0; ArchitectureDescription=The D-series Azur…
Dplsv5-Family                  @{vCPUsAvailable=0; ArchitectureDescription=The D-series Azur…
.................
Dpsv6-Family                   @{vCPUsAvailable=10; ArchitectureDescription=The D-series Azu…
#>
```

<b>NO INFORMATION ABOUT VMs</b>
```ps1
#This will run MUCH faster
Get-AzVMSku `
            -Location westeurope `
            -NoVMInformation

<# 
The output will simply have the following 3 properties empty:
1. VMSizes
2. VMSizePattern
3. VMQuotas
#>
```
[⬆️ Back to Table of Contents](#table-of-contents)

### Example 6 - Accepting Image Marketplace terms
As it is with most Azure Images, specific Azure Image terms apply which we must accept BEFORE being able to deploy using an image.

We can do this seamlessly using the helper-function that is part of the module-package.

<b>ACCEPTING MARKET-PLACE TERMS</b>
```ps1
Get-AzVmSku -Location westeurope | Set-AzVMSku

#The module will first print ALL specific terms to accept:

<# FULL OUTPUT of paloaltonetworks terms (links)
[Privacy Policy => https://www.paloaltonetworks.com/legal/privacy.html]

[License => https://storeordersprodsn.blob.core.windows.net/legalterms/3E5ED_legalterms_PALOALTONETWORKS%253a24AIRS%253a2DFLEX%253a24AIRS%253a2DBYOL%253a24T622IBUBKL6J3MHL5NUAWG2XNZ5H5FVSJGLCOC54LB63AGIONYH5CDZVDEYDONEFK2NHKCZROAP7ZU5PLZHXJ5ZNBFEUCBOWWMC4DSY.txt]

[Marketplace Terms => https://storeordersprodsn.blob.core.windows.net/marketplaceterms/3EDEF_marketplaceterms_VIRTUALMACHINE%253a24AAK2OAIZEAWW5H4MSP5KSTVB6NDKKRTUBAU23BRFTWN4YC2MQLJUB5ZEYUOUJBVF3YK34CIVPZL2HWYASPGDUY5O2FWEGRBYOXWZE5Y.txt]

Do you accept these terms? [y(yes)/n(no)]: 
#>

#Accepting terms with y gives:
<#
The agreement for URN: paloaltonetworks:airs-flex:airs-byol:11.2.501 has been accepted and the image can be used in deployments
#>
#The image can NOW be used in ANY deployment on that specific Azure subscription
```

<b>ACCEPTING TERMS WITH FORCE</b>
```ps1
#In automation scenarios, we want to skip confirmation

Get-AzVmSku -Location westeurope | Set-AzVMSku -Force

<#
Same output as above, but the module wont ask for a confirmation
#>
```
[⬆️ Back to Table of Contents](#table-of-contents)
