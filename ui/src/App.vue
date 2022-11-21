<template>
  <div class="common-layout">
    <el-container >
      <!-- 左侧 -->
      <el-aside width="200px">
        <!-- 菜单 -->
        <el-menu class="menu-layout" default-active="/" :router="true">
          <el-menu-item index="/">
            <span>首页</span>
          </el-menu-item>
          <el-menu-item index="/service">
            <span>gRPC服务</span>
          </el-menu-item>
        </el-menu>
      </el-aside>

      <!-- 右侧 -->
      <el-main>
        <RouterView></RouterView>
      </el-main>
    </el-container>
  </div>
</template>

<script>
import { RouterView } from "vue-router";

export default {
  data() {
    return {};
  },
  methods: {
    handleOpen() {},

    handleClose() {},
  },

  mounted() {},
};
</script>

<style>
.common-layout,
.container-layout {
  width: 100%;
  height: 100%;
}

.menu-layout{
  height: 100%;
}
</style>
