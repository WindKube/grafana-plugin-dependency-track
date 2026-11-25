import React from 'react';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import {
  AdvancedHttpSettings,
  Auth,
  ConfigSection,
  ConnectionSettings,
  convertLegacyAuthProps,
  DataSourceDescription,
} from '@grafana/plugin-ui';
import { config } from '@grafana/runtime';
import { Divider, Field, Input, SecretInput, SecureSocksProxySettings, Stack } from '@grafana/ui';
import { DTDataSourceOptions, DTSecureJsonData } from '../types';

type Props = DataSourcePluginOptionsEditorProps<DTDataSourceOptions, DTSecureJsonData>;

const API_KEY_METHOD = 'custom-api-key';

export function ConfigEditor({ options, onOptionsChange }: Props) {
  const { jsonData, secureJsonFields, secureJsonData } = options;
  const authProps = convertLegacyAuthProps({ config: options, onChange: onOptionsChange });

  const apiKeyInput = (
    <Field
      label="API key"
      description="Team API key or service account key with VIEW_PORTFOLIO, VIEW_VULNERABILITY and VIEW_POLICY_VIOLATION."
      required
    >
      <SecretInput
        id="config-editor-api-key"
        width={60}
        placeholder="odt_..."
        isConfigured={Boolean(secureJsonFields?.apiKey)}
        value={secureJsonData?.apiKey ?? ''}
        onChange={(e) =>
          onOptionsChange({ ...options, secureJsonData: { ...secureJsonData, apiKey: e.currentTarget.value } })
        }
        onReset={() =>
          onOptionsChange({
            ...options,
            secureJsonFields: { ...secureJsonFields, apiKey: false },
            secureJsonData: { ...secureJsonData, apiKey: '' },
          })
        }
      />
    </Field>
  );

  return (
    <Stack direction="column" gap={2}>
      <DataSourceDescription
        dataSourceName="Dependency-Track"
        docsLink="https://github.com/WindKube/grafana-plugin-dependency-track"
        hasRequiredFields
      />
      <Divider />
      <ConnectionSettings
        config={options}
        onChange={onOptionsChange}
        urlLabel="Dependency-Track API URL"
        urlPlaceholder="https://dtrack-api.example.com"
        urlTooltip="Base URL of the Dependency-Track API server, without /api. Dependency-Track 5.0 or later."
      />
      <Divider />
      <Auth
        {...authProps}
        selectedMethod={API_KEY_METHOD}
        visibleMethods={[API_KEY_METHOD]}
        customMethods={[
          {
            id: API_KEY_METHOD,
            label: 'API key',
            description: 'Authenticate with a Dependency-Track API key (X-Api-Key header).',
            component: apiKeyInput,
          },
        ]}
        onAuthMethodSelect={() => {}}
      />
      <Divider />
      <ConfigSection title="Dependency-Track UI">
        <Field
          label="UI URL"
          description="Used for links from Grafana to projects, components and vulnerabilities. Leave empty to disable links."
        >
          <Input
            id="config-editor-ui-url"
            width={60}
            placeholder="https://dtrack.example.com"
            value={jsonData.uiUrl ?? ''}
            onChange={(e) => onOptionsChange({ ...options, jsonData: { ...jsonData, uiUrl: e.currentTarget.value } })}
          />
        </Field>
      </ConfigSection>
      <Divider />
      <ConfigSection title="Additional settings" isCollapsible isInitiallyOpen={false}>
        <AdvancedHttpSettings config={options} onChange={onOptionsChange} />
        {config.secureSocksDSProxyEnabled && (
          <SecureSocksProxySettings options={options} onOptionsChange={onOptionsChange} />
        )}
      </ConfigSection>
    </Stack>
  );
}
