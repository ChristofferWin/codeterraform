<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Azure PowerShell Modules</title>
</head>

<body>

    <h1>Azure PowerShell Modules</h1>

    <p>This repository contains two PowerShell modules for managing Azure resources:</p>

    <h2>AzureVMHelper</h2>

    <p>The <code>AzureVMHelper</code> module provides functions for retrieving Azure Virtual Machine SKUs information based on specified criteria such as location, operating system, and other parameters. It can filter VM SKUs based on different settings and provide detailed information about available SKUs.</p>

    <h3>Usage Example:</h3>

    <pre><code># Retrieve Windows 10 Professional VM SKUs in East US region
Get-AzVMSKU -Location "East US" -OperatingSystem "Windows10" -OSPattern "Pro"
    </code></pre>

    <p>For more details, please refer to the <a href="./AzureVMHelper">AzureVMHelper documentation</a>.</p>

    <h2>AzureContextHelper</h2>

    <p>The <code>AzureContextHelper</code> module provides functions for setting the Azure context using advanced authentication methods. It allows users to set the Azure context using specified credentials and tenant/subscription IDs, providing interactive prompts and error handling for a seamless authentication experience.</p>

    <h3>Usage Example:</h3>

    <pre><code># Set Azure context with specified credentials and tenant/subscription IDs
Set-AzAdvancedContext -Credential (Get-Credential) -TenantID "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX" -SubscriptionID "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
    </code></pre>

    <p>For more details, please refer to the <a href="./AzureContextHelper">AzureContextHelper documentation</a>.</p>

    <h2>Prerequisites</h2>

    <ul>
        <li>PowerShell 5.1 or later</li>
        <li>Azure PowerShell module (<code>Az</code>)</li>
    </ul>

    <h2>Installation</h2>

    <ol>
        <li>Clone this repository:
            <pre><code>git clone https://github.com/yourusername/azure-powershell-modules.git
            </code></pre>
        </li>
        <li>Import the modules:
            <pre><code>Import-Module .\azure-powershell-modules\AzureVMHelper
Import-Module .\azure-powershell-modules\AzureContextHelper
            </code></pre>
        </li>
    </ol>

    <h2>Contributing</h2>

    <p>Contributions are welcome! Please feel free to open issues or submit pull requests.</p>

    <h2>License</h2>

    <p>This project is licensed under the MIT License - see the <a href="./LICENSE">LICENSE</a> file for details.</p>

</body>

</html>