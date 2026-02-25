const TOKEN_KEY = 'Authorization';
const UserProfile_KEY = 'userProfile';

function getStorageItem(key: string) {
  return localStorage.getItem(key);
}

function setStorageItem(key: string, value: string) {
  localStorage.setItem(key, value);
}

function removeStorageItem(key: string) {
  localStorage.removeItem(key);
}

function clearStorage() {
  localStorage.clear();
}

export {
  TOKEN_KEY,
  UserProfile_KEY,
  clearStorage,
  getStorageItem,
  removeStorageItem,
  setStorageItem,
};
