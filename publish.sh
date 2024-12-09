#!/bin/bash

# 定义变量
GIT_REPO_URL="https://github.com/larbing/tempfile.git"
PROJECT_DIR="/home/rock/git/tempfile"
DOCKER_CONTAINER_NAME="tempfile"
DOCKER_IMAGE_NAME="abc7223/tempfile"
DOCKER_TAG="latest" # 或者指定版本号，如 "v1.0"

# 函数：删除现有项目目录
delete_existing_project_dir() {
    echo "删除现有项目目录..."
    rm -rf $PROJECT_DIR
}


pull_git_repo() {
    echo "更新Git仓库..."
    cd $PROJECT_DIR && git pull
}

# 函数：克隆Git仓库
clone_git_repo() {
    echo "克隆Git仓库..."
    git clone $GIT_REPO_URL $PROJECT_DIR
}

# 函数：构建Docker镜像
build_docker_image() {
    echo "构建Docker镜像..."
    cd $PROJECT_DIR || exit
    docker  build -t $DOCKER_IMAGE_NAME:$DOCKER_TAG .
}

# 函数：停止并移除旧容器
stop_and_remove_old_container() {
    echo "停止并移除旧容器..."
    docker stop $DOCKER_CONTAINER_NAME && docker rm $DOCKER_CONTAINER_NAME
}

# 函数：使用新镜像启动新的容器
run_new_container() {
    echo "使用新镜像启动新的容器..."
    # 这里添加您启动容器所需的参数，例如端口映射、环境变量等
    docker run -d -p 8080:8080 --name $DOCKER_CONTAINER_NAME $DOCKER_IMAGE_NAME:$DOCKER_TAG
}

# 主程序
echo "开始更新Docker应用..."


if [ -d "$PROJECT_DIR" ]; then
    pull_git_repo
else
    clone_git_repo
fi

# 检查仓库克隆是否成功
if [ $? -eq 0 ]; then
    echo "仓库克隆成功，准备构建Docker镜像..."
    build_docker_image
else
    echo "仓库克隆失败，请检查网络连接和Git仓库URL。"
    exit 1
fi

# 检查镜像构建是否成功
if [ $? -eq 0 ]; then
    echo "Docker镜像构建成功，准备更新容器..."

    # 停止并移除旧容器
    if [ "$(docker ps -a -q -f name=$DOCKER_CONTAINER_NAME)" ]; then
        stop_and_remove_old_container
    else
        echo "未找到名为 $DOCKER_CONTAINER_NAME 的容器，将直接启动新容器..."
    fi

    # 启动新容器
    run_new_container
else
    echo "Docker镜像构建失败，请检查Dockerfile。"
    exit 1
fi

echo "Docker应用更新脚本执行完毕。"
