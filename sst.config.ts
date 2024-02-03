import { SSTConfig } from "sst";
import { Api } from "sst/constructs";

export default {
  config(_input) {
    return {
      name: "meetnearme-go-fullstack",
      region: "us-east-1",
    };
  },
  stacks(app) {
    app.setDefaultFunctionProps({
      runtime: "go",
    });
    app.stack(function Stack({ stack }) {
      const api = new Api(stack, "api", {
        routes: {
          "GET /": "functions/lambda",
          "POST /": "functions/lambda", 
        },
      });
      stack.addOutputs({
        ApiEndpoint: api.url,
      });
    });
  },
} satisfies SSTConfig;
