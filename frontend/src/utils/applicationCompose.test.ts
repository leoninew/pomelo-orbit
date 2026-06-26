import { describe, expect, it } from 'vitest';
import {
  buildEnvFile,
  buildImageCompose,
  imageCreateServiceName,
  isValidEnvKey,
} from './applicationCompose';

describe('applicationCompose', () => {
  it('builds env file content from non-empty rows', () => {
    expect(
      buildEnvFile([
        { key: ' NODE_ENV ', value: ' production ' },
        { key: '', value: '' },
        { key: 'PORT', value: '3000' },
      ])
    ).toBe('NODE_ENV=production\nPORT=3000\n');
  });

  it('builds compose with env_file when env vars exist', () => {
    expect(
      buildImageCompose({
        image: 'nginx:1.27',
        containerPort: 80,
        envVars: [{ key: 'NGINX_HOST', value: 'example.test' }],
        volumes: [],
      })
    ).toBe(`services:
  ${imageCreateServiceName}:
    image: "nginx:1.27"
    ports:
      - "80:80"
    env_file:
      - .env
`);
  });

  it('builds compose with multiple volumes', () => {
    expect(
      buildImageCompose({
        image: 'redis:7',
        containerPort: 6379,
        envVars: [],
        volumes: [
          { hostPath: './data', containerPath: '/data' },
          { hostPath: './conf', containerPath: '/usr/local/etc/redis' },
        ],
      })
    ).toBe(`services:
  ${imageCreateServiceName}:
    image: "redis:7"
    ports:
      - "6379:6379"
    volumes:
      - "./data:/data"
      - "./conf:/usr/local/etc/redis"
`);
  });

  it('omits optional env_file and volumes sections when rows are empty', () => {
    expect(
      buildImageCompose({
        image: 'busybox',
        containerPort: 8080,
        envVars: [{ key: '', value: '' }],
        volumes: [{ hostPath: '', containerPath: '' }],
      })
    ).toBe(`services:
  ${imageCreateServiceName}:
    image: "busybox"
    ports:
      - "8080:8080"
`);
  });

  it('validates dotenv keys', () => {
    expect(isValidEnvKey('APP_ENV')).toBe(true);
    expect(isValidEnvKey('_TOKEN_1')).toBe(true);
    expect(isValidEnvKey('1_TOKEN')).toBe(false);
    expect(isValidEnvKey('APP-ENV')).toBe(false);
  });
});
