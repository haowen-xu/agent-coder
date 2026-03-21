<template>
  <el-container class="auth-page">
    <el-main class="auth-content">
      <router-view />
    </el-main>
  </el-container>
</template>

<style scoped>
.auth-page {
  min-height: 100vh;
  background:
    radial-gradient(circle at 15% 20%, #edf5ff 0, transparent 42%),
    radial-gradient(circle at 85% 0, #e9f8f1 0, transparent 30%),
    linear-gradient(140deg, #f6f9fc 0%, #eef3f9 100%);
}

.auth-content {
  padding: 20px;
}
</style>
