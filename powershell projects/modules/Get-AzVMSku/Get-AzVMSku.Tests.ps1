Describe {
    It 'writes an error when NoInteractive is used without PublisherName' {
        $result = Get-AzVMSku -Location 'westeurope' -NoInteractive -ErrorAction SilentlyContinue -ErrorVariable errors

        $result | Should -BeNullOrEmpty
        $errors[0].Exception.Message | Should -Be 'You must provide a PublisherName because the switch -NoInteractive is true'
    }

    It 'writes an error when OfferName is used without PublisherName' {
        $result = Get-AzVMSku -Location 'westeurope' -OfferName 'wordpress' -ErrorAction SilentlyContinue -ErrorVariable errors

        $result | Should -BeNullOrEmpty
        $errors[0].Exception.Message | Should -Be 'A PublisherName must be provided when the OfferName is used...'
    }

    It 'returns newest SKU and newest version non-interactively' {
        $result = Get-AzVMSku `
            -Location 'westeurope' `
            -PublisherName 'bitnami' `
            -OfferName 'wordpress' `
            -NewestSKUs `
            -NewestSKUsVersions `
            -NoInteractive

        $result.Context.SubscriptionID | Should -Be 'sub-123'
        $result.Context.SubscriptionName | Should -Be 'Test Subscription'
        $result.Publisher | Should -Be 'bitnami'
        $result.Offer | Should -Be 'wordpress'
        $result.Sku | Should -Be 'wordpress-6-7'
        $result.Version | Should -Be '6.7.0'
        $result.URN | Should -Be 'bitnami:wordpress:wordpress-6-7:6.7.0'

        Should -Invoke Get-AzVMImagePublisher -Times 1 -ParameterFilter {
            $Location -eq 'westeurope'
        }

        Should -Invoke Get-AzVMImageOffer -Times 1 -ParameterFilter {
            $Location -eq 'westeurope' -and
            $PublisherName -eq 'bitnami'
        }

        Should -Invoke Get-AzVMImageSku -Times 1 -ParameterFilter {
            $Location -eq 'westeurope' -and
            $PublisherName -eq 'bitnami' -and
            $Offer -eq 'wordpress'
        }
    }

    It 'returns JSON when RawFormat is used' {
        $json = Get-AzVMSku `
            -Location 'westeurope' `
            -PublisherName 'bitnami' `
            -OfferName 'wordpress' `
            -NewestSKUs `
            -NewestSKUsVersions `
            -NoInteractive `
            -RawFormat

        $object = $json | ConvertFrom-Json

        $object.Publisher | Should -Be 'bitnami'
        $object.Offer | Should -Be 'wordpress'
        $object.SKU | Should -Be 'wordpress-6-7'
        $object.Version | Should -Be '6.7.0'
    }

}