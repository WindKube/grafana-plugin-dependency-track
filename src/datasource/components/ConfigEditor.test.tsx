import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { DataSourceSettings } from '@grafana/data';
import { ConfigEditor } from './ConfigEditor';
import { DTDataSourceOptions, DTSecureJsonData } from '../types';

const baseOptions = {
  id: 1,
  uid: 'dt',
  orgId: 1,
  name: 'Dependency-Track',
  type: 'windkube-dependencytrack-datasource',
  typeName: 'Dependency-Track',
  typeLogoUrl: '',
  access: 'proxy',
  url: 'http://dt:8080',
  user: '',
  database: '',
  basicAuth: false,
  basicAuthUser: '',
  isDefault: false,
  withCredentials: false,
  readOnly: false,
  jsonData: { uiUrl: 'http://dt-ui' },
  secureJsonFields: {},
  secureJsonData: {},
} as unknown as DataSourceSettings<DTDataSourceOptions, DTSecureJsonData>;

describe('ConfigEditor', () => {
  it('renders connection, API key and UI URL fields', () => {
    render(<ConfigEditor options={baseOptions} onOptionsChange={jest.fn()} />);
    expect(screen.getByDisplayValue('http://dt:8080')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('odt_...')).toBeInTheDocument();
    expect(screen.getByDisplayValue('http://dt-ui')).toBeInTheDocument();
  });

  it('stores the API key in secureJsonData', () => {
    const onChange = jest.fn();
    render(<ConfigEditor options={baseOptions} onOptionsChange={onChange} />);
    fireEvent.change(screen.getByPlaceholderText('odt_...'), { target: { value: 'odt_secret' } });
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ secureJsonData: { apiKey: 'odt_secret' } }));
    expect(onChange.mock.calls[0][0].jsonData).toEqual(baseOptions.jsonData);
  });

  it('stores the UI URL in jsonData', () => {
    const onChange = jest.fn();
    render(<ConfigEditor options={baseOptions} onOptionsChange={onChange} />);
    fireEvent.change(screen.getByDisplayValue('http://dt-ui'), { target: { value: 'https://dtrack.example.com' } });
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ jsonData: { uiUrl: 'https://dtrack.example.com' } })
    );
  });
});
