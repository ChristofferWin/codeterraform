#
# Module manifest for module 'Get-AzVMSku'
#
# Generated by: Christoffer Windahl Madsen
#
# Generated on: 06-10-2023
#

@{

# Script module or binary module file associated with this manifest.
RootModule = 'Get-AzVMSku.psm1'

# Version number of this module.
ModuleVersion = '2.1' 

# Supported PSEditions
# CompatiblePSEditions = @()

# ID used to uniquely identify this module
GUID = '2a85836c-f5c4-474e-ae25-d99c33d9812c'

# Author of this module
Author = 'Christoffer Windahl Madsen'

# Company or vendor of this module
CompanyName = 'Codeterraform'

# Copyright statement for this module
Copyright = '(c) Christoffer Windahl Madsen. All rights reserved.'

# Description of the functionality provided by this module
Description = 'Retrieves all required information needed in order to deploy Azure virtual machines via any IaC tool'

# Minimum version of the PowerShell engine required by this module
PowerShellVersion = '5.1'

# Name of the PowerShell host required by this module
# PowerShellHostName = ''

# Minimum version of the PowerShell host required by this module
# PowerShellHostVersion = ''

# Minimum version of Microsoft .NET Framework required by this module. This prerequisite is valid for the PowerShell Desktop edition only.
# DotNetFrameworkVersion = ''

# Minimum version of the common language runtime (CLR) required by this module. This prerequisite is valid for the PowerShell Desktop edition only.
# ClrVersion = ''

# Processor architecture (None, X86, Amd64) required by this module
# ProcessorArchitecture = ''

# Modules that must be imported into the global environment prior to importing this module
RequiredModules = @("Az.Accounts", "Az.Compute", "Az.MarketplaceOrdering", "Az.Resources")

# Assemblies that must be loaded prior to importing this module
# RequiredAssemblies = @()

# Script files (.ps1) that are run in the caller's environment prior to importing this module.
# ScriptsToProcess = @()

# Type files (.ps1xml) to be loaded when importing this module
# TypesToProcess = @()

# Format files (.ps1xml) to be loaded when importing this module
# FormatsToProcess = @()

# Modules to import as nested modules of the module specified in RootModule/ModuleToProcess
# NestedModules = @()

# Functions to export from this module, for best performance, do not use wildcards and do not delete the entry, use an empty array if there are no functions to export.
FunctionsToExport = @("Get-AzVMSKU")

# Cmdlets to export from this module, for best performance, do not use wildcards and do not delete the entry, use an empty array if there are no cmdlets to export.
CmdletsToExport = @()

# Variables to export from this module
VariablesToExport = '*'

# Aliases to export from this module, for best performance, do not use wildcards and do not delete the entry, use an empty array if there are no aliases to export.
AliasesToExport = @()

# DSC resources to export from this module
# DscResourcesToExport = @()

# List of all modules packaged with this module
# ModuleList = @()

# List of all files packaged with this module
# FileList = @()

# Private data to pass to the module specified in RootModule/ModuleToProcess. This may also contain a PSData hashtable with additional module metadata used by PowerShell.
PrivateData = @{

    PSData = @{

        # Tags applied to this module. These help with module discovery in online galleries.
        Tags = @("Azure", "ARM", "ResourceManager", "IaC", "Automation")

        # A URL to the license for this module.
        LicenseUri = 'https://github.com/ChristofferWin/codeterraform/blob/main/LICENSE'

        # A URL to the main website for this project.
        ProjectUri = 'https://codeterraform.com'

        # A URL to an icon representing this module.
        IconUri = 'https://static.wixstatic.com/media/12b015_965de78de7c74fbda9620030b81f8a1e~mv2.png/v1/fill/w_1057,h_400,al_c,q_90,usm_0.66_1.00_0.01,enc_auto/12b015_965de78de7c74fbda9620030b81f8a1e~mv2.png'

        #ReleaseNotes of this module
        ReleaseNotes = 'Due to missing release notes for version 2.0, this is included in this new release of version 2.1. For 2.0, fixed a bug where the paramter -NewestSKUsVersions resulted in an invalid result. For 2.1 fixed a bug where the error thrown for 0 skus found had invalid text.'

        # Prerelease string of this module
        # Prerelease = ''

        # Flag to indicate whether the module requires explicit user acceptance for install/update/save
        # RequireLicenseAcceptance = $false

        # External dependent modules of this module
        # ExternalModuleDependencies = @()

    } # End of PSData hashtable

} # End of PrivateData hashtable

# HelpInfo URI of this module
# HelpInfoURI = ''

# Default prefix for commands exported from this module. Override the default prefix using Import-Module -Prefix.
# DefaultCommandPrefix = ''

}

