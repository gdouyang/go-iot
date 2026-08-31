export type UserLoginType = {
  username: string
}

export type UserType = {
  username: string
  password: string
  role: Role
  roleId: string
  permissions: string[]
}

export type Role = {
  permissions: Permission[]
}

export type Permission = {
  permissionId: string
  actionEntitySet: ActionEntity[]
}

export type ActionEntity = {
  action: string
}
