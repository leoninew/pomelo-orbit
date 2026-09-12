export type EnvironmentBootstrapAuthType = 'password' | 'privateKey';

export type EnvironmentBootstrapForm = {
  username: string;
  authType: EnvironmentBootstrapAuthType;
  password: string;
  privateKey: string;
  privateKeyPassphrase: string;
};

export type EnvironmentBootstrapFormErrors = {
  username: string;
  credential: string;
};

export function emptyEnvironmentBootstrapForm(username = ''): EnvironmentBootstrapForm {
  return {
    username,
    authType: 'password',
    password: '',
    privateKey: '',
    privateKeyPassphrase: '',
  };
}

export function emptyEnvironmentBootstrapFormErrors(): EnvironmentBootstrapFormErrors {
  return { username: '', credential: '' };
}

export function validateEnvironmentBootstrapForm(
  form: EnvironmentBootstrapForm,
  messages: { username: string; password: string; privateKey: string }
): EnvironmentBootstrapFormErrors {
  return {
    username: form.username.trim() && !/[\r\n]/.test(form.username) ? '' : messages.username,
    credential:
      form.authType === 'password'
        ? form.password
          ? ''
          : messages.password
        : form.privateKey.trim()
          ? ''
          : messages.privateKey,
  };
}

export function environmentBootstrapRequestFromForm(form: EnvironmentBootstrapForm) {
  return {
    username: form.username.trim(),
    password: form.authType === 'password' ? form.password : '',
    private_key: form.authType === 'privateKey' ? form.privateKey : '',
    private_key_passphrase: form.authType === 'privateKey' ? form.privateKeyPassphrase : '',
  };
}
