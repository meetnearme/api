import { Function, use } from 'sst/constructs';
import envVars from './shared/env';
import { StorageStack } from './StorageStack';

export function SeshuFunction({ stack }: StackContext) {
  const { seshuSessionsTable } = use(StorageStack);

  const seshuFn = new Function(stack, 'SeshuFunction', {
    handler: 'functions/lambda/go/seshu',
    runtime: 'go',
    url: true,
    timeout: 10 * 60, // seconds
    memorySize: 512,
    environment: {
      ...envVars,
    },
    bind: [seshuSessionsTable],
  });
  stack.addOutputs({
    SeshuFunctionUrl: seshuFn.url,
  });
  return { seshuFn };
}
