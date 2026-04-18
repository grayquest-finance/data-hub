properties([
    parameters([
        string(
            name: 'BRANCH_NAME',
            defaultValue: 'release_uat',
            description: 'The release branch to deploy (e.g., release_uat, release_staging, release_prod)'
        ),
        string(
            name: 'COMMIT_ID',
            defaultValue: '',
            description: 'Specific commit ID to deploy (optional)'
        ),
        string(
            name: 'TRIGGERED_BY',
            defaultValue: 'manual',
            description: 'Who/what triggered this deployment'
        ),
        choice(
            name: 'DEPLOYMENT_MODE',
            choices: ['standard', 'force', 'rollback'],
            description: 'Deployment mode: standard (normal), force (skip validations), rollback (revert to previous)'
        ),
    ]),
    disableConcurrentBuilds(),
    buildDiscarder(logRotator(numToKeepStr: '30'))
])

def extractEnvironment(branchName) {
    def lower = branchName.toLowerCase()
    if (lower.contains('uat'))     return 'uat'
    if (lower.contains('staging')) return 'staging'
    if (lower.contains('preprod')) return 'preprod'
    return 'uat'
}

def getClusterName(environment) {
    switch(environment) {
        case 'uat':     return 'sandbox-tools'
        case 'staging': return 'sandbox-tools'
        case 'preprod': return 'pre-prod-cluster'
        case 'prod':    return 'production-cluster'
        default:        return 'sandbox-tools'
    }
}

pipeline {
    agent { label 'gq_arm_' }

    environment {
        AWS_DEFAULT_REGION   = 'ap-south-1'
        ECR_REGISTRY         = '579897422692.dkr.ecr.ap-south-1.amazonaws.com'

        TARGET_ENVIRONMENT   = extractEnvironment(params.BRANCH_NAME)
        ECR_REPOSITORY_NAME  = "data_hub_${TARGET_ENVIRONMENT}"
        EKS_CLUSTER_NAME     = getClusterName(TARGET_ENVIRONMENT)
        K8S_NAMESPACE        = "data-hub-${TARGET_ENVIRONMENT}"
        K8S_DEPLOYMENT       = 'data-hub'

        APP_NAME             = 'data-hub'
        TARGET_BRANCH        = "${params.BRANCH_NAME}"
        COMMIT_ID            = "${params.COMMIT_ID ?: env.GIT_COMMIT?.take(8) ?: 'latest'}"
        IMAGE_TAG            = "${TARGET_BRANCH}-${COMMIT_ID}"
        FULL_IMAGE_URI       = "${ECR_REGISTRY}/${ECR_REPOSITORY_NAME}:${IMAGE_TAG}"

        DEPLOYMENT_MODE      = "${params.DEPLOYMENT_MODE}"
        TRIGGERED_BY         = "${params.TRIGGERED_BY}"
        MAX_RELEASES_TO_KEEP = '20'
    }

    stages {
        stage('Initialize') {
            steps {
                script {
                    echo "🚀 STARTING DATA HUB DEPLOYMENT"
                    echo "═══════════════════════════════════════"
                    echo "📦 Application:  ${APP_NAME}"
                    echo "🌿 Branch:       ${TARGET_BRANCH}"
                    echo "📝 Commit:       ${COMMIT_ID}"
                    echo "🏷️  Image Tag:   ${IMAGE_TAG}"
                    echo "⚙️  Mode:         ${DEPLOYMENT_MODE}"
                    echo "👤 Triggered by: ${TRIGGERED_BY}"
                    echo "☸️  EKS Cluster: ${EKS_CLUSTER_NAME}"
                    echo "📁 Namespace:    ${K8S_NAMESPACE}"
                    echo "═══════════════════════════════════════"
                }
            }
        }

        stage('Checkout') {
            steps {
                script {
                    checkout([
                        $class: 'GitSCM',
                        branches: [[name: "origin/${TARGET_BRANCH}"]],
                        doGenerateSubmoduleConfigurations: false,
                        extensions: [
                            [$class: 'CleanBeforeCheckout'],
                            [$class: 'CloneOption', depth: 1, shallow: true]
                        ],
                        submoduleCfg: [],
                        userRemoteConfigs: scm.userRemoteConfigs
                    ])

                    if (params.COMMIT_ID && params.COMMIT_ID.trim() != '') {
                        env.COMMIT_ID = params.COMMIT_ID.trim()
                    } else {
                        env.COMMIT_ID = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
                    }

                    env.IMAGE_TAG      = "${TARGET_BRANCH}-${env.COMMIT_ID}"
                    env.FULL_IMAGE_URI = "${ECR_REGISTRY}/${ECR_REPOSITORY_NAME}:${env.IMAGE_TAG}"

                    echo "📝 Commit: ${env.COMMIT_ID}"
                    echo "🏷️  Tag:    ${env.IMAGE_TAG}"
                }
            }
        }

        stage('Validate') {
            steps {
                sh '''
                    [ -f Dockerfile ]  || (echo "❌ Dockerfile not found" && exit 1)
                    [ -f Makefile ]    || (echo "❌ Makefile not found" && exit 1)
                    [ -f go.mod ]      || (echo "❌ go.mod not found" && exit 1)
                    [ -f cmd/main.go ] || (echo "❌ cmd/main.go not found" && exit 1)
                    echo "✅ All required files found"
                '''
            }
        }

        stage('Build Docker Image') {
            when {
                not { equals expected: 'rollback', actual: params.DEPLOYMENT_MODE }
            }
            steps {
                sh '''
                    make docker-build

                    docker tag ${APP_NAME}:latest ${APP_NAME}:${IMAGE_TAG}
                    docker tag ${APP_NAME}:${IMAGE_TAG} ${ECR_REGISTRY}/${ECR_REPOSITORY_NAME}:${IMAGE_TAG}

                    docker images | grep ${APP_NAME} | head -3
                    echo "✅ Image built: ${IMAGE_TAG}"
                '''
            }
        }

        stage('Login to ECR') {
            when {
                not { equals expected: 'rollback', actual: params.DEPLOYMENT_MODE }
            }
            steps {
                sh '''
                    make ecr-login
                    echo "✅ ECR login successful"
                '''
            }
        }

        stage('Push to ECR') {
            when {
                not { equals expected: 'rollback', actual: params.DEPLOYMENT_MODE }
            }
            steps {
                sh '''
                    make ecr-push ENVIRONMENT=${TARGET_ENVIRONMENT} IMAGE_TAG=${IMAGE_TAG}

                    aws ecr describe-images \
                        --repository-name ${ECR_REPOSITORY_NAME} \
                        --image-ids imageTag=${IMAGE_TAG} \
                        --region ${AWS_DEFAULT_REGION}
                    echo "✅ Image pushed to ECR"
                '''
            }
        }

        stage('Set kubectl Context') {
            steps {
                sh '''
                    make eks-context ENVIRONMENT=${TARGET_ENVIRONMENT}
                    kubectl get nodes
                    echo "✅ kubectl context set"
                '''
            }
        }

        stage('Rollback') {
            when {
                equals expected: 'rollback', actual: params.DEPLOYMENT_MODE
            }
            steps {
                sh '''
                    make eks-rollback ENVIRONMENT=${TARGET_ENVIRONMENT}
                    echo "✅ Rollback initiated"
                '''
            }
        }

        stage('Deploy to EKS') {
            when {
                not { equals expected: 'rollback', actual: params.DEPLOYMENT_MODE }
            }
            steps {
                sh '''
                    make eks-deploy ENVIRONMENT=${TARGET_ENVIRONMENT} IMAGE_TAG=${IMAGE_TAG}
                    echo "✅ Deployment initiated"
                '''
            }
        }

        stage('Health Check') {
            steps {
                script {
                    timeout(time: 10, unit: 'MINUTES') {
                        sh '''
                            make eks-wait ENVIRONMENT=${TARGET_ENVIRONMENT}
                            echo "✅ Rollout complete"
                        '''
                    }
                    sh 'make eks-status ENVIRONMENT=${TARGET_ENVIRONMENT}'
                }
            }
        }

        stage('Cleanup Old ECR Images') {
            when {
                not { equals expected: 'rollback', actual: params.DEPLOYMENT_MODE }
            }
            steps {
                sh '''
                    make ecr-cleanup ENVIRONMENT=${TARGET_ENVIRONMENT} MAX_RELEASES=${MAX_RELEASES_TO_KEEP}
                    echo "✅ Cleanup complete"
                '''
            }
        }
    }

    post {
        always {
            sh '''
                docker rmi ${APP_NAME}:${IMAGE_TAG} 2>/dev/null || true
                docker rmi ${ECR_REGISTRY}/${ECR_REPOSITORY_NAME}:${IMAGE_TAG} 2>/dev/null || true
                docker images --filter "reference=${APP_NAME}" --filter "dangling=true" -q \
                    | xargs -r docker rmi 2>/dev/null || true
            '''
        }

        success {
            script {
                def type = (DEPLOYMENT_MODE == 'rollback') ? 'ROLLBACK' : 'DEPLOYMENT'
                echo "🎉 DATA HUB ${type} SUCCESSFUL!"
                echo "✅ Branch: ${TARGET_BRANCH} | Commit: ${COMMIT_ID} | Cluster: ${EKS_CLUSTER_NAME}/${K8S_NAMESPACE}"
            }
        }

        failure {
            script {
                def type = (DEPLOYMENT_MODE == 'rollback') ? 'ROLLBACK' : 'DEPLOYMENT'
                echo "❌ DATA HUB ${type} FAILED! Build: ${env.BUILD_URL}"
            }
        }

        aborted {
            echo "⏹️ Deployment aborted — Branch: ${TARGET_BRANCH}, Triggered by: ${TRIGGERED_BY}"
        }
    }
}