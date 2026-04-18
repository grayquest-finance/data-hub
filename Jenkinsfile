pipeline {
    agent {
        kubernetes {
            yaml """
apiVersion: v1
kind: Pod
spec:
  serviceAccountName: jenkins-service-account
  containers:
  - name: jenkins-agent
    image: ${params.JENKINS_AGENT_IMAGE}
    tty: true
    command: ["cat"]
    volumeMounts:
    - mountPath: /home/jenkins/agent
      name: workspace-volume
    env:
    - name: AWS_DEFAULT_REGION
      value: "ap-south-1"
  - name: aws-cli
    image: amazon/aws-cli:2.15.17
    tty: true
    command: ["cat"]
    volumeMounts:
    - mountPath: /home/jenkins/agent
      name: workspace-volume
    env:
    - name: AWS_DEFAULT_REGION
      value: "ap-south-1"
  - name: kaniko
    image: gcr.io/kaniko-project/executor:debug
    tty: true
    command: ["/busybox/cat"]
    volumeMounts:
    - mountPath: /home/jenkins/agent
      name: workspace-volume
    env:
    - name: AWS_DEFAULT_REGION
      value: "ap-south-1"
    resources:
      requests:
        cpu: "500m"
        memory: "2Gi"
        ephemeral-storage: "4Gi"
      limits:
        cpu: "1"
        memory: "4Gi"
        ephemeral-storage: "4Gi"

  - name: helm-deploy
    image: dtzar/helm-kubectl:3.12.3
    tty: true
    command: ["sleep", "3600"]
    volumeMounts:
    - mountPath: /home/jenkins/agent
      name: workspace-volume
    env:
    - name: AWS_DEFAULT_REGION
      value: "ap-south-1"
  volumes:
  - name: workspace-volume
    emptyDir: {}
            """
        }
    }
    environment {
        AWS_ACCOUNT_ID    = "${params.AWS_ACCOUNT_ID}"
        AWS_REGION        = "${params.AWS_REGION}"
        ECR_REPO          = "${params.ECR_REPO}"
        APP_NAME          = "${params.APP_NAME}"
        SONARQUBE_ENV     = "SonarQube"
        APP_RELEASE       = "${params.APP_RELEASE}"
        RUN_SONAR         = "${params.RUN_SONAR}"
        EKS_CLUSTER_NAME  = "${params.EKS_CLUSTER_NAME}"
        NAMESPACE         = "${params.NAMESPACE}"
    }
    options {
        timeout(time: 1, unit: 'HOURS')
    }
    stages {
        stage('Print Parameters') {
            steps {
                container('jenkins-agent') {
                    sh '''
                        echo "===== BUILD PARAMETERS ====="
                        echo "AWS_REGION        = ${AWS_REGION}"
                        echo "EKS_CLUSTER_NAME  = ${EKS_CLUSTER_NAME}"
                        echo "NAMESPACE         = ${NAMESPACE}"
                        echo "ECR_REPO          = ${ECR_REPO}"
                        echo "APP_NAME          = ${APP_NAME}"
                        echo "GIT_BRANCH        = ${GIT_BRANCH}"
                        echo "============================"
                    '''
                }
            }
        }
        stage('Clone Code') {
            steps {
                container('jenkins-agent') {
                    git branch: "${params.GIT_BRANCH}",
                        url: 'https://github.com/shravan-amberkar-gq/data-hub.git',
                        credentialsId: 'test-cred-git'

                    sh '''
                        echo "Repository cloned successfully"
                        ls -la
                        head -n 20 Dockerfile
                    '''
                }
            }
        }

        stage('Set Image Tag') {
            steps {
                container('jenkins-agent') {
                    script {
                        def branch = sh(script: "git rev-parse --abbrev-ref HEAD", returnStdout: true).trim()
                        branch = branch.replaceAll('/', '-')
                        def shortCommit = sh(script: "git rev-parse --short=7 HEAD", returnStdout: true).trim()

                        env.IMAGE_TAG = "${branch}-${shortCommit}"
                        env.LATEST_TAG = "latest"

                        echo "Tags for this build: ${IMAGE_TAG}, ${LATEST_TAG}"
                    }
                }
            }
        }

        stage('SonarQube Scan') {
            when { expression { return env.RUN_SONAR == "true" } }
            steps {
                container('jenkins-agent') {
                    withSonarQubeEnv("${SONARQUBE_ENV}") {
                        sh '''
                            mvn clean verify sonar:sonar \
                            -Dsonar.projectKey=${APP_NAME} \
                            -Dsonar.projectName=${APP_NAME} \
                            -Dsonar.projectVersion=${IMAGE_TAG}
                        '''
                    }
                }
            }
        }

        stage('Check Tools') {
            steps {
                script {
                    parallel(
                        "Check AWS CLI": {
                            container('aws-cli') {
                                sh '''
                                    echo "Checking AWS CLI..."
                                    aws --version
                                    aws sts get-caller-identity
                                    echo "AWS CLI OK"
                                '''
                            }
                        },
                        "Check Kaniko": {
                            container('kaniko') {
                                sh '''
                                    echo "Checking Kaniko..."
                                    ls -la /kaniko
                                    echo "Kaniko OK"
                                '''
                            }
                        },
                        "Check Helm & Kubectl": {
                            container('helm-deploy') {
                                sh '''
                                    echo "Checking Helm..."
                                    helm version
                                    echo "Checking Kubectl..."
                                    kubectl version --client=true
                                '''
                            }
                        }
                    )
                }
            }
        }

        stage('Prepare ECR Authentication') {
            steps {
                container('aws-cli') {
                    sh '''
                        echo "Setting up ECR authentication for Kaniko..."
                        mkdir -p /home/jenkins/agent/.docker
                        REG="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
                        PW=$(aws ecr get-login-password --region ${AWS_REGION})
                        AUTH=$(printf "AWS:%s" "$PW" | base64 -w0)
                        cat > /home/jenkins/agent/.docker/config.json <<EOF
{"auths":{"$REG":{"auth":"$AUTH"}}}
EOF
                        echo "ECR authentication configured successfully"
                    '''
                }
            }
        }


        stage('Build & Push Image with Kaniko') {
            steps {
                container('kaniko') {
                    withCredentials([string(credentialsId: 'GitHub-Token', variable: 'GITHUB_TOKEN')]) {
                        sh '''
                            REG="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
                            echo "Pushing to $REG with tags ${IMAGE_TAG} and ${LATEST_TAG}"

                            /kaniko/executor \
                                --context ${WORKSPACE} \
                                --dockerfile ${WORKSPACE}/Dockerfile \
                                --destination ${REG}/${ECR_REPO}:${IMAGE_TAG} \
                                --destination ${REG}/${ECR_REPO}:${LATEST_TAG} \
                                --build-arg GITHUB_TOKEN=${GITHUB_TOKEN} \
                                --cache=true \
                                --verbosity=info
                        '''
                    }
                }
            }
        }

        stage('Setup Kubernetes Access') {
            steps {
                container('aws-cli') {
                    sh '''
                        echo "Setting up Kubernetes access..."
                        aws --version
                        aws sts get-caller-identity
                        aws eks update-kubeconfig --region ${AWS_REGION} --name ${EKS_CLUSTER_NAME}
                    '''
                }
            }
        }

        stage('Deploy with Helm') {
            steps {
                container('helm-deploy') {
                    sh '''
                        REG="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
                        echo "Deploying application using Helm..."
                        helm upgrade --install ${APP_RELEASE} ./helm \
                            --namespace ${NAMESPACE} --create-namespace \
                            --set image.repository=${REG}/${ECR_REPO} \
                            --set image.tag=${IMAGE_TAG} \
                            --set app.name=${APP_NAME} \
                            -f helm/values.yaml

                        echo "Waiting for rollout..."
                        kubectl rollout status deployment/${APP_RELEASE} -n ${NAMESPACE} --timeout=300s
                        kubectl get pods -l app=${APP_RELEASE} -n ${NAMESPACE}
                    '''
                }
            }
        }
    }

    post {
        always {
            sh '''
                echo "Deployment process completed."
            '''
        }
        success {
            echo """
✅ Pipeline succeeded!
Image: ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}:${IMAGE_TAG}
App:   ${APP_RELEASE}
Namespace: ${NAMESPACE}
            """
        }
        failure {
            echo """
❌ Pipeline failed!
Check:
1. EKS_CLUSTER_NAME is correct
2. IAM role has ECR + EKS access
3. Cluster connectivity
4. Helm chart path exists (./helm-chart)
            """
        }
    }
}
