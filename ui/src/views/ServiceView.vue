<template>
  <el-form :inline="true" :model="service" class="demo-form-inline">
    <el-form-item label="名称">
      <el-input v-model="service.name" placeholder="名称" type="text" />
    </el-form-item>

    <el-form-item label="主机" :rules="[
      { required: true, message: 'host is required' },
    ]">
      <el-input v-model="service.host" placeholder="127.0.0.1" type="text" />
    </el-form-item>

    <el-form-item label="端口" :rules="[
      { required: true, message: 'port is required' },
      { type: 'number', message: 'port must be a number' },
    ]">
      <el-input v-model.number="service.port" type="text" />
    </el-form-item>

    <el-form-item>
      <el-button type="primary" @click="addService">添加</el-button>
    </el-form-item>

  </el-form>
</template>


<script>

import axios from "axios";
import { ElNotification } from "element-plus";



export default {
  data() {
    return {
      service: {
        host: '127.0.0.1',
        port: 0
      }
    };
  },

  methods: {

    addService() {
      console.table(this.service);

      let param = this.service;
      
      axios
          .post(`/rpc/services`, param)
          .then(function (response) {
            // console.log(response.data);
            let result = response.data;

            pThis.tabSelect = "tabItemRes"

            pThis.jsonRpcData = {
              header: result.header,
              trailer: result.trailer,
            }
            pThis.jsonData = result.data;
          })
          .catch(function (error) {
            console.log(error);
            ElNotification({
              title: "Warning",
              message: JSON.stringify(error.response.data.error, null, 2),
              type: "warning",
              position: "bottom-right",
            });
          })
          .then(function () {
            // 总是会执行
          });

    }

  },

  mounted() {

  }


};


</script>


<style>

</style>
