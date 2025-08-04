#!/usr/bin/env node
/**
 * Generate .env file from JSON configuration.
 * This script reads env-vars.json and creates a .env file with the appropriate values.
 * Designed to be called directly from GitHub Actions workflows.
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

function loadConfig() {
  /** Load the environment variables configuration from JSON. */
  const __filename = fileURLToPath(import.meta.url);
  const __dirname = path.dirname(__filename);
  const configPath = path.join(__dirname, 'env-vars.json');
  const configData = fs.readFileSync(configPath, 'utf8');
  return JSON.parse(configData);
}

function maskSensitiveValue(value) {
  /** Mask sensitive values for logging - show only first and last character with exactly 3 asterisks. */
  if (!value || value.length <= 2) {
    return value;
  }
  return `${value.charAt(0)}***${value.charAt(value.length - 1)}`;
}

function generateEnvFile(config, deploymentValues = {}, stage = 'dev') {
  /** Generate .env file content. */
  let envContent = '';

  // Add deployment variables first
  console.log('\n🔧 Adding deployment variables:');
  for (const [varName, varConfig] of Object.entries(config.deployment_vars)) {
    if (deploymentValues[varName]) {
      envContent += `${varName}=${deploymentValues[varName]}\n`;
      console.log(
        `  ✅ ${varName}=${maskSensitiveValue(
          deploymentValues[varName],
        )} (deployment)`,
      );
    }
  }

  // Create a map of resolved environment variables with de-duplication logic
  const resolvedVars = new Map();

  // First pass: collect all environment variables
  for (const varName of Object.keys(config.env_vars)) {
    if (process.env[varName]) {
      resolvedVars.set(varName, process.env[varName]);
    }
  }

  // Second pass: handle stage-specific variable de-duplication
  const stagePrefix = stage === 'prod' ? '_PROD_' : '_DEV_';
  let overrideCount = 0;

  for (const [varName, value] of resolvedVars.entries()) {
    // Check if this variable has a stage-specific prefix
    if (varName.startsWith(stagePrefix)) {
      // Extract the base variable name (remove the prefix)
      const baseVarName = varName.substring(stagePrefix.length);

      // If we have both the prefixed and non-prefixed version,
      // the prefixed version takes precedence
      if (resolvedVars.has(baseVarName)) {
        const baseValue = resolvedVars.get(baseVarName);
        overrideCount++;
        console.log(
          `🔄 Variable override: ${baseVarName} (${maskSensitiveValue(
            baseValue,
          )}) → ${varName} (${maskSensitiveValue(
            value,
          )}) (${stage.toUpperCase()} override)`,
        );
        resolvedVars.delete(baseVarName);
      }
    }
  }

  if (overrideCount > 0) {
    console.log(
      `\n📊 Stage-specific overrides applied: ${overrideCount} variable(s)`,
    );
  }

  // Third pass: write the resolved variables to .env
  console.log(
    `\n📝 Writing ${resolvedVars.size} environment variables to .env file:`,
  );
  for (const [varName, value] of resolvedVars.entries()) {
    // For stage-specific variables, write them as the base variable name
    let finalVarName = varName;
    let overrideNote = '';
    if (varName.startsWith(stagePrefix)) {
      finalVarName = varName.substring(stagePrefix.length);
      overrideNote = ` (${stage.toUpperCase()} override from ${varName})`;
    }

    if (finalVarName === 'USER_TEAM_EMAIL_SCHEMA') {
      // Special handling for quoted values
      envContent += `${finalVarName}="${value}"\n`;
      console.log(
        `  ✅ ${finalVarName}="${maskSensitiveValue(value)}"${overrideNote}`,
      );
    } else {
      envContent += `${finalVarName}=${value}\n`;
      console.log(
        `  ✅ ${finalVarName}=${maskSensitiveValue(value)}${overrideNote}`,
      );
    }
  }

  return envContent;
}

// Main execution
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);

  if (args.includes('--help')) {
    console.log('Usage:');
    console.log('  node generate-env-inline.js --dev');
    console.log('  node generate-env-inline.js --prod');
    console.log('');
    console.log(
      'This will generate a .env file from env-vars.json configuration',
    );
    process.exit(0);
  }

  const config = loadConfig();

  // Define deployment-specific values based on environment
  let deploymentValues = {
    USE_REMOTE_DB: 'true',
    IS_LOCAL_ACT: 'false',
    DEPLOYMENT_TARGET: 'ACT',
    WEAVIATE_SCHEME: 'http',
    WEAVIATE_HOST: 'weaviate',
    WEAVIATE_PORT: '8080',
    NATS_URL: 'nats://nats-server:4222',
    NATS_SESHU_STREAM_NAME: 'SESHU_JOBS_STREAM',
    NATS_SESHU_STREAM_SUBJECT: 'seshu.jobs.queue',
    NATS_SESHU_STREAM_DURABLE_NAME: 'seshu-consume',
  };

  if (args.includes('--dev')) {
    deploymentValues = {
      ...deploymentValues,
      ACT_STAGE: 'dev',
      SST_Table_tableName_CompetitionConfig:
        'dev-meetnearme-go-fullstack-CompetitionConfig',
      SST_Table_tableName_CompetitionRounds:
        'dev-meetnearme-go-fullstack-CompetitionRounds',
      SST_Table_tableName_CompetitionWaitingRoomParticipant:
        'dev-meetnearme-go-fullstack-CompetitionWaitingRoomParticipantmParticipant',
      SST_Table_tableName_EventRsvps: 'dev-meetnearme-go-fullstack-EventRsvps',
      SST_Table_tableName_Purchasables:
        'dev-meetnearme-go-fullstack-Purchasables',
      SST_Table_tableName_PurchasesV2:
        'dev-meetnearme-go-fullstack-PurchasesV2',
      SST_Table_tableName_RegistrationFields:
        'dev-meetnearme-go-fullstack-RegistrationFields',
      SST_Table_tableName_Registrations:
        'dev-meetnearme-go-fullstack-Registrations',
      SST_Table_tableName_SeshuSessions:
        'dev-meetnearme-go-fullstack-SeshuSessions',
      SST_Table_tableName_Votes: 'dev-meetnearme-go-fullstack-Votes',
    };
  } else if (args.includes('--prod')) {
    deploymentValues = {
      ...deploymentValues,
      ACT_STAGE: 'prod',
      SST_Table_tableName_CompetitionConfig:
        'prod-meetnearme-go-fullstack-CompetitionConfig',
      SST_Table_tableName_CompetitionRounds:
        'prod-meetnearme-go-fullstack-CompetitionRounds',
      SST_Table_tableName_CompetitionWaitingRoomParticipant:
        'prod-meetnearme-go-fullstack-CompetitionWaitingRoomParticipantmParticipant',
      SST_Table_tableName_EventRsvps: 'prod-meetnearme-go-fullstack-EventRsvps',
      SST_Table_tableName_Purchasables:
        'prod-meetnearme-go-fullstack-Purchasables',
      SST_Table_tableName_PurchasesV2:
        'prod-meetnearme-go-fullstack-PurchasesV2',
      SST_Table_tableName_RegistrationFields:
        'prod-meetnearme-go-fullstack-RegistrationFields',
      SST_Table_tableName_Registrations:
        'prod-meetnearme-go-fullstack-Registrations',
      SST_Table_tableName_SeshuSessions:
        'prod-meetnearme-go-fullstack-SeshuSessions',
      SST_Table_tableName_Votes: 'prod-meetnearme-go-fullstack-Votes',
    };
  }

  const stage = args.includes('--prod') ? 'prod' : 'dev';
  console.log(`🚀 Generating .env file for ${stage.toUpperCase()} environment`);
  console.log(`🔍 Stage prefix: ${stage === 'prod' ? '_PROD_' : '_DEV_'}\n`);

  const envContent = generateEnvFile(config, deploymentValues, stage);

  // Write .env file
  fs.writeFileSync('.env', envContent);
  console.log('Generated .env file from configuration');

  // Validate that all required variables are present
  const missingVars = [];
  const stagePrefix = stage === 'prod' ? '_PROD_' : '_DEV_';

  // Create a map of available variables for validation
  const availableVars = new Map();
  for (const varName of Object.keys(config.env_vars)) {
    if (process.env[varName]) {
      availableVars.set(varName, process.env[varName]);
    }
  }

  // Handle stage-specific variable de-duplication for validation
  for (const [varName, value] of availableVars.entries()) {
    if (varName.startsWith(stagePrefix)) {
      const baseVarName = varName.substring(stagePrefix.length);
      // If we have a stage-specific variable, it satisfies the base variable requirement
      // So we add the base variable name to availableVars if it's not already there
      if (!availableVars.has(baseVarName)) {
        availableVars.set(baseVarName, value);
      }
      // Remove the stage-specific variable from the list since it's now represented by the base name
      availableVars.delete(varName);
    }
  }

  // Check environment variables
  for (const [varName, varConfig] of Object.entries(config.env_vars)) {
    if (varConfig.required) {
      // Skip stage-specific variables in validation - they're handled separately
      if (varName.startsWith('_DEV_') || varName.startsWith('_PROD_')) {
        continue;
      }

      // Check if the base variable is available
      if (availableVars.has(varName)) {
        continue; // Base variable is available
      }

      // Check if a stage-specific version is available for the current stage
      const stageSpecificVar = `${stagePrefix}${varName}`;
      if (availableVars.has(stageSpecificVar)) {
        continue; // Stage-specific variable is available
      }

      // If neither base nor stage-specific is available, it's missing
      missingVars.push(varName);
    }
  }

  // Check deployment variables
  for (const [varName, varConfig] of Object.entries(config.deployment_vars)) {
    if (
      varConfig.required &&
      !availableVars.has(varName) &&
      !deploymentValues[varName]
    ) {
      missingVars.push(varName);
    }
  }

  if (missingVars.length > 0) {
    console.error('❌ Missing required environment variables:');
    missingVars.forEach((varName) => console.error(`  - ${varName}`));
    process.exit(1);
  }

  console.log('✅ All required environment variables are present');
}

export { loadConfig, generateEnvFile };
