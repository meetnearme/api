import { Function } from 'sst/constructs';
import envVars from './shared/env';

export function SeshuFunction({ stack }: StackContext) {
  const seshuFn = new Function(stack, 'SeshuFunction', {
    handler: 'functions/lambda/go/seshu',
    runtime: 'go',
    url: true,
    timeout: 10 * 60, // seconds
    memorySize: 512,
    environment: {
      ...envVars,
    },
  });
  stack.addOutputs({
    SeshuFunctionUrl: seshuFn.url,
  });
  return { seshuFn };
}
