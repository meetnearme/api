import { StackContext } from 'sst/constructs';
import * as ec2 from "aws-cdk-lib/aws-ec2";
import * as ecs from "aws-cdk-lib/aws-ecs";
import * as lb from "aws-cdk-lib/aws-elasticloadbalancingv2"
import * as autoscaling from "aws-cdk-lib/aws-autoscaling";
import { Duration } from "aws-cdk-lib";

export function MarqoStack({ stack }: StackContext) {
    // Create a VPC
    const vpc = new ec2.Vpc(stack, "MarqoVpc", {
        maxAzs: 1  // Use only one AZ
    });

    // Create an ECS cluster
    const cluster = new ecs.Cluster(stack, "MarqoCluster", {
        vpc: vpc
    });

    // Create an EC2 instance for the ECS cluster
    const autoScalingGroup = new autoscaling.AutoScalingGroup(stack, "MarqoASG", {
        vpc: vpc,
        instanceType: ec2.InstanceType.of(ec2.InstanceClass.C7G, ec2.InstanceSize.XLARGE4),  // 16 vCPUs, 32 GB RAM
        machineImage: ecs.EcsOptimizedImage.amazonLinux2(),
        minCapacity: 1,
        maxCapacity: 1,
        desiredCapacity: 1,
        blockDevices: [
            {
                deviceName: "/dev/xvda",
                volume: autoscaling.BlockDeviceVolume.ebs(500)  // 500 GB EBS volume
            }
        ]
    });

    // Add the EC2 instance to the ECS cluster
    const capacityProvider = new ecs.AsgCapacityProvider(stack, "MarqoAsgCapacityProvider", {
        autoScalingGroup: autoScalingGroup
    });
    cluster.addAsgCapacityProvider(capacityProvider);

    // Create a task definition
    const taskDefinition = new ecs.Ec2TaskDefinition(stack, "MarqoTaskDefinition");

    // Add container to task definition
    const container = taskDefinition.addContainer("MarqoContainer", {
        image: ecs.ContainerImage.fromRegistry("marqoai/marqo:2.11"),
        memoryReservationMiB: 30000,  // Reserve 30 GB of memory
        cpu: 15360,  // Use 15 vCPUs (leaving some capacity for the EC2 instance itself)
        portMappings: [{ containerPort: 8882 }],
        logging: ecs.LogDrivers.awsLogs({ streamPrefix: "marqo" }),
    });

    // Add a volume to the task definition
    taskDefinition.addVolume({
        name: "marqo-data",
    });

    // Mount the volume to the container
    container.addMountPoints({
        sourceVolume: "marqo-data",
        containerPath: "/opt/var/vespa",
        readOnly: false
    });

    // Create an ECS service
    const marqoService = new ecs.Ec2Service(stack, "MarqoService", {
        cluster: cluster,
        taskDefinition: taskDefinition,
        desiredCount: 1,
        capacityProviderStrategies: [
            {
                capacityProvider: capacityProvider.capacityProviderName,
                weight: 1
            }
        ]
    });

    const loadBalancer = new lb.ApplicationLoadBalancer(stack, 'MarqoALB', {
        loadBalancerName: "marqo-alb",
        vpc: vpc,
        internetFacing: false,
        vpcSubnets: { subnetType: ec2.SubnetType.PRIVATE_ISOLATED, availabilityZones: ['us-east-1a', 'us-east-1b'] }
    });

    const listener = loadBalancer.addListener('Listener', {
        port: 80,
    });

    // Create a target group
    const targetGroup = listener.addTargets('ECS', {
        port: 8882,
        protocol: lb.ApplicationProtocol.HTTP,
        targets: [marqoService],
        healthCheck: {
            path: '/health',
            interval: Duration.minutes(5),
            timeout: Duration.seconds(10)
        }
    });

    marqoService.connections.allowFrom(loadBalancer, ec2.Port.tcp(8882));


    // Output the API URL
    stack.addOutputs({
        "MarqoAlbDnsName": loadBalancer.loadBalancerDnsName,
        "MarqoAlbArn": loadBalancer.loadBalancerArn
    });
    return {marqoService};
}
