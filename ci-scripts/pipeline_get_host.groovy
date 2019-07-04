#!/usr/bin/env groovy

def projectName = env.JOB_NAME
def groupId = 'se.su.it'
def goGitPath = 'github.com/stockholmuniversity'

def revision
def tag

def ldflags

suSetProperties(["github": "true"])

node("agent") {
    stage("Cleanup workspace")
    {
        cleanWs()
    }

    stage("Prepare docker environment")
    {
        suDockerBuildAndPull(projectName)
    }

    docker.image(projectName).inside('-v /local/jenkins/conf:/local/jenkins/conf -v /local/jenkins/libexec:/local/jenkins/libexec') {

        suGitHubBuildStatus {
            def goPath = sh(script: 'echo $GOPATH', returnStdout: true).trim()
            def srcDir = "${goPath}/src/${goGitPath}"
            stage("Prepare build")
            {
                sh "mkdir -p ${srcDir}"

                suCheckoutCode ([
                    projectName: projectName,
                    targetDir: "checkout",
                ])

                sh "mv ${WORKSPACE}/checkout ${srcDir}/${projectName}"
            }

            stage("Get information")
            {
                revision = env.rev ?: sh(script: "git log -n 1  --pretty=format:'%H'", returnStdout: true).trim()
                tag = sh(script: "git tag --contains ${revision} | tail -1", returnStdout: true).trim()
            }

            stage("Build")
            {
                if (tag) {
                    ldflags = "-ldflags \"-X main.version=${tag}\""
                } else {
                    ldflags = "-ldflags \"-X main.version=${revision}\""
                }
                sh "cd ${srcDir}/${projectName} && go build -tags=netgo ${ldflags}"
            }

            stage("Create archive")
            {
                sh "tar -czvf ${projectName}.tgz -C ${srcDir}/${projectName} ${projectName}"
            }

            stage("Deploy to Nexus")
            {
                if (tag) {
                    sh "/local/jenkins/libexec/deploy-to-nexus --file ${projectName}.tgz --project ${projectName} --groupId ${groupId} --commitType tag --version ${tag}"
                }
            }
        }
    }
}
